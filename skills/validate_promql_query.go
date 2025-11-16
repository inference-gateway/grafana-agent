package skills

import (
	"context"
	"encoding/json"
	"fmt"

	server "github.com/inference-gateway/adk/server"
	promql "github.com/inference-gateway/grafana-agent/internal/promql"
	zap "go.uber.org/zap"
)

// ValidatePromqlQuerySkill struct holds the skill with services
type ValidatePromqlQuerySkill struct {
	logger *zap.Logger
	promql promql.PromQL
}

// NewValidatePromqlQuerySkill creates a new validate_promql_query skill
func NewValidatePromqlQuerySkill(logger *zap.Logger, promql promql.PromQL) server.Tool {
	skill := &ValidatePromqlQuerySkill{
		logger: logger,
		promql: promql,
	}
	return server.NewBasicTool(
		"validate_promql_query",
		"Validates a PromQL query against a Prometheus server",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"prometheus_url": map[string]any{
					"description": "Prometheus server URL to validate against",
					"type":        "string",
				},
				"query": map[string]any{
					"description": "PromQL query to validate",
					"type":        "string",
				},
			},
			"required": []string{"prometheus_url", "query"},
		},
		skill.ValidatePromqlQueryHandler,
	)
}

// ValidateQueryResponse represents the validation result
type ValidateQueryResponse struct {
	PrometheusURL string `json:"prometheus_url"`
	Query         string `json:"query"`
	Valid         bool   `json:"valid"`
	Error         string `json:"error,omitempty"`
}

// ValidatePromqlQueryHandler handles the validate_promql_query skill execution
func (s *ValidatePromqlQuerySkill) ValidatePromqlQueryHandler(ctx context.Context, args map[string]any) (string, error) {
	s.logger.Info("validating promql query")

	prometheusURL, ok := args["prometheus_url"].(string)
	if !ok || prometheusURL == "" {
		return "", fmt.Errorf("prometheus_url is required and must be a string")
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("query is required and must be a string")
	}

	s.logger.Debug("validating query",
		zap.String("query", query),
		zap.String("prometheus_url", prometheusURL))

	response := ValidateQueryResponse{
		PrometheusURL: prometheusURL,
		Query:         query,
		Valid:         false,
	}

	// Validate the query
	err := s.promql.ValidateQuery(ctx, prometheusURL, query)
	if err != nil {
		s.logger.Warn("query validation failed",
			zap.String("query", query),
			zap.Error(err))
		response.Error = err.Error()
		response.Valid = false
	} else {
		s.logger.Info("query validation succeeded",
			zap.String("query", query))
		response.Valid = true
	}

	// Marshal response to JSON
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(jsonData), nil
}
