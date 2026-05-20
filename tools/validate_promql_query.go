package tools

import (
	"context"
	"encoding/json"
	"fmt"

	server "github.com/inference-gateway/adk/server"
	promql "github.com/inference-gateway/grafana-agent/internal/promql"
	zap "go.uber.org/zap"
)

// ValidatePromqlQueryTool struct holds the tool with services
type ValidatePromqlQueryTool struct {
	logger *zap.Logger
	promql promql.PromQL
}

// NewValidatePromqlQueryTool creates a new validate_promql_query tool
func NewValidatePromqlQueryTool(logger *zap.Logger, promql promql.PromQL) server.Tool {
	tool := &ValidatePromqlQueryTool{
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
		tool.ValidatePromqlQueryHandler,
	)
}

// ValidateQueryResponse represents the validation result
type ValidateQueryResponse struct {
	PrometheusURL string `json:"prometheus_url"`
	Query         string `json:"query"`
	Valid         bool   `json:"valid"`
	Error         string `json:"error,omitempty"`
}

// ValidatePromqlQueryHandler handles the validate_promql_query tool execution
func (t *ValidatePromqlQueryTool) ValidatePromqlQueryHandler(ctx context.Context, args map[string]any) (string, error) {
	t.logger.Info("validating promql query")

	prometheusURL, ok := args["prometheus_url"].(string)
	if !ok || prometheusURL == "" {
		return "", fmt.Errorf("prometheus_url is required and must be a string")
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("query is required and must be a string")
	}

	t.logger.Debug("validating query",
		zap.String("query", query),
		zap.String("prometheus_url", prometheusURL))

	response := ValidateQueryResponse{
		PrometheusURL: prometheusURL,
		Query:         query,
		Valid:         false,
	}

	err := t.promql.ValidateQuery(ctx, prometheusURL, query)
	if err != nil {
		t.logger.Warn("query validation failed",
			zap.String("query", query),
			zap.Error(err))
		response.Error = err.Error()
		response.Valid = false
	} else {
		t.logger.Info("query validation succeeded",
			zap.String("query", query))
		response.Valid = true
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(jsonData), nil
}
