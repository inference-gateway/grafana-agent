package skills

import (
	"context"
	"encoding/json"
	"fmt"

	server "github.com/inference-gateway/adk/server"
	promql "github.com/inference-gateway/grafana-agent/internal/promql"
	zap "go.uber.org/zap"
)

// DiscoverMetricsSkill struct holds the skill with services
type DiscoverMetricsSkill struct {
	logger *zap.Logger
	promql promql.PromQL
}

// NewDiscoverMetricsSkill creates a new discover_metrics skill
func NewDiscoverMetricsSkill(logger *zap.Logger, promql promql.PromQL) server.Tool {
	skill := &DiscoverMetricsSkill{
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
		skill.DiscoverMetricsHandler,
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

// DiscoverMetricsHandler handles the discover_metrics skill execution
func (s *DiscoverMetricsSkill) DiscoverMetricsHandler(ctx context.Context, args map[string]any) (string, error) {
	s.logger.Info("discovering metrics")

	// Extract prometheus_url (required)
	prometheusURL, ok := args["prometheus_url"].(string)
	if !ok || prometheusURL == "" {
		return "", fmt.Errorf("prometheus_url is required and must be a string")
	}

	// Extract optional name_pattern
	namePattern := ""
	if pattern, ok := args["name_pattern"].(string); ok {
		namePattern = pattern
	}

	// Extract optional metric_type
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

	s.logger.Debug("discovering metrics with filters",
		zap.String("prometheus_url", prometheusURL),
		zap.String("name_pattern", namePattern),
		zap.String("metric_type", metricTypeStr))

	// Discover metrics
	metrics, err := s.promql.DiscoverMetrics(ctx, prometheusURL, namePattern, metricType)
	if err != nil {
		s.logger.Error("failed to discover metrics",
			zap.String("prometheus_url", prometheusURL),
			zap.Error(err))
		return "", fmt.Errorf("failed to discover metrics: %w", err)
	}

	// Build response
	response := DiscoverMetricsResponse{
		PrometheusURL: prometheusURL,
		TotalMetrics:  len(metrics),
		Metrics:       metrics,
	}

	// Add filter information if filters were applied
	if namePattern != "" || metricTypeStr != "" {
		response.Filters = FilterInfo{
			NamePattern: namePattern,
			MetricType:  metricTypeStr,
		}
	}

	s.logger.Info("discovered metrics",
		zap.String("prometheus_url", prometheusURL),
		zap.Int("total", len(metrics)))

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(jsonData), nil
}
