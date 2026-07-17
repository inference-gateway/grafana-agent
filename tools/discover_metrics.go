package tools

import (
	"context"
	"encoding/json"
	"fmt"

	zap "go.uber.org/zap"

	server "github.com/inference-gateway/adk/server"

	promql "github.com/inference-gateway/grafana-agent/internal/promql"
)

// DiscoverMetricsTool struct holds the tool with services
type DiscoverMetricsTool struct {
	logger *zap.Logger
	promql promql.PromQL
}

// NewDiscoverMetricsTool creates a new discover_metrics tool
func NewDiscoverMetricsTool(logger *zap.Logger, promql promql.PromQL) server.Tool {
	tool := &DiscoverMetricsTool{
		logger: logger,
		promql: promql,
	}
	return server.NewBasicTool(
		"discover_metrics",
		"Discovers available metrics from a Prometheus endpoint with optional filtering",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"metric_type": map[string]any{
					"description": "Optional metric type filter (counter, gauge, histogram, summary)",
					"enum":        []string{"counter", "gauge", "histogram", "summary"},
					"type":        "string",
				},
				"name_pattern": map[string]any{
					"description": "Optional regex pattern to filter metrics by name",
					"type":        "string",
				},
				"prometheus_url": map[string]any{
					"description": "Prometheus server URL to discover metrics from",
					"type":        "string",
				},
			},
			"required": []string{"prometheus_url"},
		},
		tool.DiscoverMetricsHandler,
	)
}

// DiscoverMetricsResponse represents the response from metric discovery
type DiscoverMetricsResponse struct {
	PrometheusURL string              `json:"prometheus_url"`
	TotalMetrics  int                 `json:"total_metrics"`
	Metrics       []promql.MetricInfo `json:"metrics"`
	Filters       FilterInfo          `json:"filters,omitempty"`
}

// FilterInfo contains information about applied filters
type FilterInfo struct {
	NamePattern string `json:"name_pattern,omitempty"`
	MetricType  string `json:"metric_type,omitempty"`
}

// DiscoverMetricsHandler handles the discover_metrics tool execution
func (t *DiscoverMetricsTool) DiscoverMetricsHandler(ctx context.Context, args map[string]any) (string, error) {
	span := startToolSpan(ctx, "discover_metrics")
	defer span.End()

	t.logger.Info("discovering metrics")

	prometheusURL, ok := args["prometheus_url"].(string)
	if !ok || prometheusURL == "" {
		return "", fmt.Errorf("prometheus_url is required and must be a string")
	}

	namePattern := ""
	if pattern, ok := args["name_pattern"].(string); ok {
		namePattern = pattern
	}

	metricTypeStr := ""
	var metricType promql.MetricType
	if mt, ok := args["metric_type"].(string); ok {
		metricTypeStr = mt
		switch mt {
		case "counter":
			metricType = promql.MetricTypeCounter
		case "gauge":
			metricType = promql.MetricTypeGauge
		case "histogram":
			metricType = promql.MetricTypeHistogram
		case "summary":
			metricType = promql.MetricTypeSummary
		default:
			metricType = promql.MetricTypeUnknown
		}
	}

	t.logger.Debug("discovering metrics with filters",
		zap.String("prometheus_url", prometheusURL),
		zap.String("name_pattern", namePattern),
		zap.String("metric_type", metricTypeStr))

	metrics, err := t.promql.DiscoverMetrics(ctx, prometheusURL, namePattern, metricType)
	if err != nil {
		t.logger.Error("failed to discover metrics",
			zap.String("prometheus_url", prometheusURL),
			zap.Error(err))
		return "", fmt.Errorf("failed to discover metrics: %w", err)
	}

	response := DiscoverMetricsResponse{
		PrometheusURL: prometheusURL,
		TotalMetrics:  len(metrics),
		Metrics:       metrics,
	}

	if namePattern != "" || metricTypeStr != "" {
		response.Filters = FilterInfo{
			NamePattern: namePattern,
			MetricType:  metricTypeStr,
		}
	}

	t.logger.Info("discovered metrics",
		zap.String("prometheus_url", prometheusURL),
		zap.Int("total", len(metrics)))

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(jsonData), nil
}
