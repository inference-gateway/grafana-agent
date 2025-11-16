package skills

import (
	"context"
	"encoding/json"
	"fmt"

	server "github.com/inference-gateway/adk/server"
	promql "github.com/inference-gateway/grafana-agent/internal/promql"
	zap "go.uber.org/zap"
)

// GeneratePromqlQueriesSkill struct holds the skill with services
type GeneratePromqlQueriesSkill struct {
	logger *zap.Logger
	promql promql.PromQL
}

// NewGeneratePromqlQueriesSkill creates a new generate_promql_queries skill
func NewGeneratePromqlQueriesSkill(logger *zap.Logger, promql promql.PromQL) server.Tool {
	skill := &GeneratePromqlQueriesSkill{
		logger: logger,
		promql: promql,
	}
	return server.NewBasicTool(
		"generate_promql_queries",
		"Generates PromQL query suggestions for given metric names by querying Prometheus metadata",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"metric_names": map[string]any{
					"description": "Array of metric names to generate queries for",
					"items":       map[string]any{"type": "string"},
					"type":        "array",
				},
				"prometheus_url": map[string]any{
					"description": "Prometheus server URL for querying metric metadata",
					"type":        "string",
				},
			},
			"required": []string{"prometheus_url", "metric_names"},
		},
		skill.GeneratePromqlQueriesHandler,
	)
}

// QueryGenerationResult represents the result for a single metric
type QueryGenerationResult struct {
	MetricName  string                   `json:"metric_name"`
	MetricType  string                   `json:"metric_type"`
	MetricHelp  string                   `json:"metric_help"`
	Labels      []string                 `json:"labels,omitempty"`
	Suggestions []promql.QuerySuggestion `json:"suggestions"`
	Error       string                   `json:"error,omitempty"`
}

// GeneratePromqlQueriesResponse represents the overall response
type GeneratePromqlQueriesResponse struct {
	PrometheusURL string                  `json:"prometheus_url"`
	Results       []QueryGenerationResult `json:"results"`
}

// GeneratePromqlQueriesHandler handles the generate_promql_queries skill execution
func (s *GeneratePromqlQueriesSkill) GeneratePromqlQueriesHandler(ctx context.Context, args map[string]any) (string, error) {
	s.logger.Info("generating promql queries")

	prometheusURL, ok := args["prometheus_url"].(string)
	if !ok || prometheusURL == "" {
		return "", fmt.Errorf("prometheus_url is required and must be a string")
	}

	metricNamesRaw, ok := args["metric_names"]
	if !ok {
		return "", fmt.Errorf("metric_names is required")
	}

	metricNamesSlice, ok := metricNamesRaw.([]any)
	if !ok {
		return "", fmt.Errorf("metric_names must be an array")
	}

	if len(metricNamesSlice) == 0 {
		return "", fmt.Errorf("metric_names cannot be empty")
	}

	metricNames := make([]string, 0, len(metricNamesSlice))
	for _, mn := range metricNamesSlice {
		if metricName, ok := mn.(string); ok {
			metricNames = append(metricNames, metricName)
		}
	}

	response := GeneratePromqlQueriesResponse{
		PrometheusURL: prometheusURL,
		Results:       make([]QueryGenerationResult, 0, len(metricNames)),
	}

	for _, metricName := range metricNames {
		s.logger.Debug("processing metric", zap.String("metric", metricName))

		result := QueryGenerationResult{
			MetricName: metricName,
		}

		metricInfo, err := s.promql.GetMetricMetadata(ctx, prometheusURL, metricName)
		if err != nil {
			s.logger.Warn("failed to get metric metadata",
				zap.String("metric", metricName),
				zap.Error(err))
			result.Error = fmt.Sprintf("failed to get metadata: %v", err)
			response.Results = append(response.Results, result)
			continue
		}

		result.MetricType = string(metricInfo.Type)
		result.MetricHelp = metricInfo.Help
		result.Labels = metricInfo.Labels

		suggestions := s.promql.GenerateQueries(metricInfo)
		if len(suggestions) == 0 {
			s.logger.Warn("no suggestions generated",
				zap.String("metric", metricName))
			result.Error = "no query suggestions could be generated"
			response.Results = append(response.Results, result)
			continue
		}

		result.Suggestions = suggestions
		response.Results = append(response.Results, result)

		s.logger.Info("generated queries for metric",
			zap.String("metric", metricName),
			zap.Int("suggestion_count", len(suggestions)))
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(jsonData), nil
}
