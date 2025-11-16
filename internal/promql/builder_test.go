package promql

import (
	"testing"
)

func TestInferMetricType(t *testing.T) {
	tests := []struct {
		name       string
		metricName string
		expected   MetricType
	}{
		{
			name:       "counter with _total suffix",
			metricName: "http_requests_total",
			expected:   MetricTypeCounter,
		},
		{
			name:       "counter with _count in name",
			metricName: "request_count",
			expected:   MetricTypeCounter,
		},
		{
			name:       "histogram with _bucket suffix",
			metricName: "http_duration_bucket",
			expected:   MetricTypeHistogram,
		},
		{
			name:       "gauge with memory in name",
			metricName: "memory_usage_bytes",
			expected:   MetricTypeGauge,
		},
		{
			name:       "gauge with cpu in name",
			metricName: "cpu_usage_percent",
			expected:   MetricTypeGauge,
		},
		{
			name:       "unknown metric type",
			metricName: "random_metric",
			expected:   MetricTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferMetricType(tt.metricName)
			if result != tt.expected {
				t.Errorf("inferMetricType(%s) = %s, want %s", tt.metricName, result, tt.expected)
			}
		})
	}
}

func TestGenerateCounterQueries(t *testing.T) {
	metricInfo := &MetricInfo{
		Name:   "http_requests_total",
		Type:   MetricTypeCounter,
		Help:   "Total HTTP requests",
		Labels: []string{"method", "status", "__name__"},
	}

	suggestions := generateCounterQueries(metricInfo)

	if len(suggestions) < 2 {
		t.Errorf("Expected at least 2 suggestions, got %d", len(suggestions))
	}

	foundRate := false
	foundIncrease := false
	for _, suggestion := range suggestions {
		if suggestion.Query == "rate(http_requests_total[5m])" {
			foundRate = true
			if suggestion.VisualizationType != "timeseries" {
				t.Errorf("Rate query should have timeseries visualization, got %s", suggestion.VisualizationType)
			}
		}
		if suggestion.Query == "increase(http_requests_total[1h])" {
			foundIncrease = true
		}
	}

	if !foundRate {
		t.Error("Expected rate query not found")
	}
	if !foundIncrease {
		t.Error("Expected increase query not found")
	}
}

func TestGenerateGaugeQueries(t *testing.T) {
	metricInfo := &MetricInfo{
		Name:   "memory_usage_bytes",
		Type:   MetricTypeGauge,
		Help:   "Memory usage in bytes",
		Labels: []string{"instance", "__name__"},
	}

	suggestions := generateGaugeQueries(metricInfo)

	if len(suggestions) < 3 {
		t.Errorf("Expected at least 3 suggestions, got %d", len(suggestions))
	}

	found := false
	for _, suggestion := range suggestions {
		if suggestion.Query == "memory_usage_bytes" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected basic gauge query not found")
	}
}

func TestGenerateHistogramQueries(t *testing.T) {
	metricInfo := &MetricInfo{
		Name: "http_duration_bucket",
		Type: MetricTypeHistogram,
		Help: "HTTP request duration",
	}

	suggestions := generateHistogramQueries(metricInfo)

	if len(suggestions) < 3 {
		t.Errorf("Expected at least 3 suggestions, got %d", len(suggestions))
	}

	// Check for histogram_quantile queries
	foundQuantile := false
	for _, suggestion := range suggestions {
		if suggestion.Query == "histogram_quantile(0.50, rate(http_duration_bucket[5m]))" {
			foundQuantile = true
			if suggestion.VisualizationType != "timeseries" {
				t.Errorf("Quantile query should have timeseries visualization, got %s", suggestion.VisualizationType)
			}
		}
	}

	if !foundQuantile {
		t.Error("Expected 50th percentile query not found")
	}
}

func TestGetBestQuery(t *testing.T) {
	suggestions := []QuerySuggestion{
		{
			Query:             "rate(metric[5m])",
			Description:       "Rate per second",
			VisualizationType: "timeseries",
		},
		{
			Query:             "increase(metric[1h])",
			Description:       "Total increase",
			VisualizationType: "stat",
		},
	}

	best := getBestQuery(suggestions)
	if best.Query != "rate(metric[5m])" {
		t.Errorf("Expected first query as best, got %s", best.Query)
	}

	// Test empty suggestions
	empty := getBestQuery([]QuerySuggestion{})
	if empty.Query != "up" {
		t.Errorf("Expected default 'up' query for empty suggestions, got %s", empty.Query)
	}
}

func TestPrometheusClientValidateQuery(t *testing.T) {
	client := newPrometheusClient("http://localhost:9090")

	if client.baseURL != "http://localhost:9090" {
		t.Errorf("Expected baseURL to be http://localhost:9090, got %s", client.baseURL)
	}

	clientWithSlash := newPrometheusClient("http://localhost:9090/")
	if clientWithSlash.baseURL != "http://localhost:9090" {
		t.Errorf("Expected trailing slash to be trimmed, got %s", clientWithSlash.baseURL)
	}
}

func TestMetricInfoCreation(t *testing.T) {
	metricInfo := &MetricInfo{
		Name:   "test_metric",
		Type:   MetricTypeGauge,
		Help:   "Test metric description",
		Labels: []string{"label1", "label2"},
	}

	if metricInfo.Name != "test_metric" {
		t.Errorf("Expected Name to be 'test_metric', got %s", metricInfo.Name)
	}

	if metricInfo.Type != MetricTypeGauge {
		t.Errorf("Expected Type to be MetricTypeGauge, got %s", metricInfo.Type)
	}

	if len(metricInfo.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(metricInfo.Labels))
	}
}

func TestQuerySuggestionCreation(t *testing.T) {
	suggestion := QuerySuggestion{
		Query:             "up",
		Description:       "Service up status",
		VisualizationType: "stat",
		YAxisLabel:        "status",
	}

	if suggestion.Query != "up" {
		t.Errorf("Expected Query to be 'up', got %s", suggestion.Query)
	}

	if suggestion.VisualizationType != "stat" {
		t.Errorf("Expected VisualizationType to be 'stat', got %s", suggestion.VisualizationType)
	}
}

// Benchmark tests for performance verification
func BenchmarkGenerateCounterQueries(b *testing.B) {
	metricInfo := &MetricInfo{
		Name:   "http_requests_total",
		Type:   MetricTypeCounter,
		Labels: []string{"method", "status"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateCounterQueries(metricInfo)
	}
}

func BenchmarkInferMetricType(b *testing.B) {
	metricNames := []string{
		"http_requests_total",
		"memory_usage_bytes",
		"http_duration_bucket",
		"cpu_usage_percent",
		"random_metric",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, name := range metricNames {
			inferMetricType(name)
		}
	}
}
