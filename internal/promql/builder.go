package promql

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MetricType represents the type of a Prometheus metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
	MetricTypeUnknown   MetricType = "unknown"
)

// MetricInfo represents metadata about a Prometheus metric
type MetricInfo struct {
	Name   string     `json:"name"`
	Type   MetricType `json:"type"`
	Help   string     `json:"help"`
	Labels []string   `json:"labels"`
}

// QuerySuggestion represents a suggested PromQL query for a metric
type QuerySuggestion struct {
	Query             string `json:"query"`
	Description       string `json:"description"`
	VisualizationType string `json:"visualization_type"`
	YAxisLabel        string `json:"y_axis_label"`
}

// prometheusClient handles communication with Prometheus API
type prometheusClient struct {
	baseURL string
	client  *http.Client
}

// newPrometheusClient creates a new Prometheus client
func newPrometheusClient(baseURL string) *prometheusClient {
	return &prometheusClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// getMetricMetadata fetches metadata for a specific metric from Prometheus
func (c *prometheusClient) getMetricMetadata(ctx context.Context, metricName string) (*MetricInfo, error) {
	metadataURL := fmt.Sprintf("%s/api/v1/metadata?metric=%s", c.baseURL, url.QueryEscape(metricName))

	req, err := http.NewRequestWithContext(ctx, "GET", metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query Prometheus metadata: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prometheus returned status %d", resp.StatusCode)
	}

	var metadataResp struct {
		Status string `json:"status"`
		Data   map[string][]struct {
			Type MetricType `json:"type"`
			Help string     `json:"help"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&metadataResp); err != nil {
		return nil, fmt.Errorf("failed to decode metadata response: %w", err)
	}

	if metadataResp.Status != "success" {
		return nil, fmt.Errorf("prometheus API returned non-success status: %s", metadataResp.Status)
	}

	data, exists := metadataResp.Data[metricName]
	if !exists || len(data) == 0 {
		inferredType := inferMetricType(metricName)
		return &MetricInfo{
			Name: metricName,
			Type: inferredType,
			Help: "No metadata available",
		}, nil
	}

	labels, err := c.getMetricLabels(ctx, metricName)
	if err != nil {
		labels = []string{}
	}

	return &MetricInfo{
		Name:   metricName,
		Type:   data[0].Type,
		Help:   data[0].Help,
		Labels: labels,
	}, nil
}

// getMetricLabels fetches available labels for a metric
func (c *prometheusClient) getMetricLabels(ctx context.Context, metricName string) ([]string, error) {
	labelsURL := fmt.Sprintf("%s/api/v1/labels", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", labelsURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get labels: status %d", resp.StatusCode)
	}

	var labelsResp struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&labelsResp); err != nil {
		return nil, err
	}

	if labelsResp.Status != "success" {
		return nil, fmt.Errorf("labels API returned non-success status: %s", labelsResp.Status)
	}

	return labelsResp.Data, nil
}

// validateQuery validates a PromQL query against Prometheus
func (c *prometheusClient) validateQuery(ctx context.Context, query string) error {
	queryURL := fmt.Sprintf("%s/api/v1/query", c.baseURL)

	data := url.Values{}
	data.Set("query", query)
	data.Set("time", "0") // Use epoch time for validation

	req, err := http.NewRequestWithContext(ctx, "POST", queryURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create validation request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate query: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var queryResp struct {
		Status    string `json:"status"`
		Error     string `json:"error"`
		ErrorType string `json:"errorType"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
		return fmt.Errorf("failed to decode validation response: %w", err)
	}

	if queryResp.Status != "success" {
		return fmt.Errorf("query validation failed: %s (%s)", queryResp.Error, queryResp.ErrorType)
	}

	return nil
}

// generateQueries generates appropriate PromQL queries based on metric type and name
func generateQueries(metricInfo *MetricInfo) []QuerySuggestion {
	var suggestions []QuerySuggestion

	switch metricInfo.Type {
	case MetricTypeCounter:
		suggestions = generateCounterQueries(metricInfo)
	case MetricTypeGauge:
		suggestions = generateGaugeQueries(metricInfo)
	case MetricTypeHistogram:
		suggestions = generateHistogramQueries(metricInfo)
	case MetricTypeSummary:
		suggestions = generateSummaryQueries(metricInfo)
	default:
		suggestions = generateDefaultQueries(metricInfo)
	}

	return suggestions
}

// generateCounterQueries generates queries for counter metrics
func generateCounterQueries(metricInfo *MetricInfo) []QuerySuggestion {
	metricName := metricInfo.Name

	suggestions := []QuerySuggestion{
		{
			Query:             fmt.Sprintf("rate(%s[5m])", metricName),
			Description:       "Rate per second over 5 minutes",
			VisualizationType: "timeseries",
			YAxisLabel:        "per second",
		},
		{
			Query:             fmt.Sprintf("increase(%s[1h])", metricName),
			Description:       "Total increase over 1 hour",
			VisualizationType: "timeseries",
			YAxisLabel:        "total",
		},
	}

	if len(metricInfo.Labels) > 0 {
		for _, label := range metricInfo.Labels {
			if label != "__name__" && !strings.HasPrefix(label, "__") {
				suggestions = append(suggestions, QuerySuggestion{
					Query:             fmt.Sprintf("sum by (%s) (rate(%s[5m]))", label, metricName),
					Description:       fmt.Sprintf("Rate per second grouped by %s", label),
					VisualizationType: "timeseries",
					YAxisLabel:        "per second",
				})
			}
		}
	}

	return suggestions
}

// generateGaugeQueries generates queries for gauge metrics
func generateGaugeQueries(metricInfo *MetricInfo) []QuerySuggestion {
	metricName := metricInfo.Name

	suggestions := []QuerySuggestion{
		{
			Query:             metricName,
			Description:       "Current value",
			VisualizationType: "timeseries",
			YAxisLabel:        "value",
		},
		{
			Query:             fmt.Sprintf("avg_over_time(%s[1h])", metricName),
			Description:       "Average over 1 hour",
			VisualizationType: "timeseries",
			YAxisLabel:        "avg value",
		},
	}

	if len(metricInfo.Labels) > 0 {
		suggestions = append(suggestions,
			QuerySuggestion{
				Query:             fmt.Sprintf("avg(%s)", metricName),
				Description:       "Average across all instances",
				VisualizationType: "stat",
				YAxisLabel:        "avg value",
			},
			QuerySuggestion{
				Query:             fmt.Sprintf("max(%s)", metricName),
				Description:       "Maximum value",
				VisualizationType: "stat",
				YAxisLabel:        "max value",
			},
			QuerySuggestion{
				Query:             fmt.Sprintf("min(%s)", metricName),
				Description:       "Minimum value",
				VisualizationType: "stat",
				YAxisLabel:        "min value",
			},
		)

		for _, label := range metricInfo.Labels {
			if label != "__name__" && !strings.HasPrefix(label, "__") {
				suggestions = append(suggestions, QuerySuggestion{
					Query:             fmt.Sprintf("avg by (%s) (%s)", label, metricName),
					Description:       fmt.Sprintf("Average grouped by %s", label),
					VisualizationType: "timeseries",
					YAxisLabel:        "avg value",
				})
			}
		}
	}

	return suggestions
}

// generateHistogramQueries generates queries for histogram metrics
func generateHistogramQueries(metricInfo *MetricInfo) []QuerySuggestion {
	baseName := strings.TrimSuffix(metricInfo.Name, "_bucket")
	baseName = strings.TrimSuffix(baseName, "_count")
	baseName = strings.TrimSuffix(baseName, "_sum")

	suggestions := []QuerySuggestion{
		{
			Query:             fmt.Sprintf("histogram_quantile(0.50, rate(%s_bucket[5m]))", baseName),
			Description:       "50th percentile (median) over 5 minutes",
			VisualizationType: "timeseries",
			YAxisLabel:        "duration",
		},
		{
			Query:             fmt.Sprintf("histogram_quantile(0.95, rate(%s_bucket[5m]))", baseName),
			Description:       "95th percentile over 5 minutes",
			VisualizationType: "timeseries",
			YAxisLabel:        "duration",
		},
		{
			Query:             fmt.Sprintf("histogram_quantile(0.99, rate(%s_bucket[5m]))", baseName),
			Description:       "99th percentile over 5 minutes",
			VisualizationType: "timeseries",
			YAxisLabel:        "duration",
		},
		{
			Query:             fmt.Sprintf("rate(%s_count[5m])", baseName),
			Description:       "Request rate (requests per second)",
			VisualizationType: "timeseries",
			YAxisLabel:        "requests/sec",
		},
		{
			Query:             fmt.Sprintf("rate(%s_sum[5m]) / rate(%s_count[5m])", baseName, baseName),
			Description:       "Average duration",
			VisualizationType: "timeseries",
			YAxisLabel:        "avg duration",
		},
	}

	return suggestions
}

// generateSummaryQueries generates queries for summary metrics
func generateSummaryQueries(metricInfo *MetricInfo) []QuerySuggestion {
	baseName := strings.TrimSuffix(metricInfo.Name, "_count")
	baseName = strings.TrimSuffix(baseName, "_sum")

	suggestions := []QuerySuggestion{
		{
			Query:             fmt.Sprintf("rate(%s_count[5m])", baseName),
			Description:       "Request rate (requests per second)",
			VisualizationType: "timeseries",
			YAxisLabel:        "requests/sec",
		},
		{
			Query:             fmt.Sprintf("rate(%s_sum[5m]) / rate(%s_count[5m])", baseName, baseName),
			Description:       "Average value",
			VisualizationType: "timeseries",
			YAxisLabel:        "avg value",
		},
	}

	// Add quantile queries if available
	if strings.Contains(metricInfo.Name, "_count") || strings.Contains(metricInfo.Name, "_sum") {
		// Try common quantiles
		for _, quantile := range []string{"0.5", "0.9", "0.95", "0.99"} {
			suggestions = append(suggestions, QuerySuggestion{
				Query:             fmt.Sprintf("%s{quantile=\"%s\"}", baseName, quantile),
				Description:       fmt.Sprintf("%s quantile", quantile),
				VisualizationType: "timeseries",
				YAxisLabel:        "value",
			})
		}
	}

	return suggestions
}

// generateDefaultQueries generates default queries for unknown metric types
func generateDefaultQueries(metricInfo *MetricInfo) []QuerySuggestion {
	metricName := metricInfo.Name

	if strings.HasSuffix(metricName, "_total") ||
		strings.Contains(metricName, "_count") ||
		strings.Contains(metricName, "requests") ||
		strings.Contains(metricName, "errors") {
		return generateCounterQueries(metricInfo)
	}

	return []QuerySuggestion{
		{
			Query:             metricName,
			Description:       "Raw metric value",
			VisualizationType: "timeseries",
			YAxisLabel:        "value",
		},
		{
			Query:             fmt.Sprintf("rate(%s[5m])", metricName),
			Description:       "Rate of change over 5 minutes",
			VisualizationType: "timeseries",
			YAxisLabel:        "per second",
		},
	}
}

// inferMetricType attempts to infer the metric type from the metric name
func inferMetricType(metricName string) MetricType {
	if strings.HasSuffix(metricName, "_total") ||
		strings.Contains(metricName, "_count") ||
		strings.Contains(metricName, "requests") ||
		strings.Contains(metricName, "errors") {
		return MetricTypeCounter
	}

	if strings.Contains(metricName, "_bucket") ||
		strings.Contains(metricName, "_duration") ||
		strings.Contains(metricName, "_latency") {
		return MetricTypeHistogram
	}

	if strings.Contains(metricName, "size") ||
		strings.Contains(metricName, "usage") ||
		strings.Contains(metricName, "memory") ||
		strings.Contains(metricName, "cpu") {
		return MetricTypeGauge
	}

	return MetricTypeUnknown
}

// getBestQuery selects the most appropriate query for visualization
func getBestQuery(suggestions []QuerySuggestion) QuerySuggestion {
	if len(suggestions) == 0 {
		return QuerySuggestion{
			Query:             "up",
			Description:       "Default query",
			VisualizationType: "timeseries",
			YAxisLabel:        "value",
		}
	}

	return suggestions[0]
}
