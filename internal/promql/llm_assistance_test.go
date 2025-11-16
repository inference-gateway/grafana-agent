package promql

import (
	"context"
	"testing"
)

func TestLLMQueryEnhancer(t *testing.T) {
	enhancer := NewLLMQueryEnhancer()
	
	if enhancer == nil {
		t.Error("Expected non-nil LLM query enhancer")
	}
}

func TestEnhanceQueries(t *testing.T) {
	enhancer := NewLLMQueryEnhancer()
	
	metricInfo := &MetricInfo{
		Name: "http_requests_total",
		Type: MetricTypeCounter,
		Help: "Total HTTP requests",
	}
	
	suggestions := []QuerySuggestion{
		{
			Query:             "rate(http_requests_total[5m])",
			Description:       "Rate per second over 5 minutes",
			VisualizationType: "timeseries",
			YAxisLabel:        "per second",
		},
	}
	
	enhanced := enhancer.EnhanceQueries(context.Background(), metricInfo, suggestions)
	
	// Should have at least the original suggestion plus contextual ones
	if len(enhanced) < len(suggestions) {
		t.Errorf("Enhanced suggestions should be >= original, got %d vs %d", len(enhanced), len(suggestions))
	}
}

func TestEnhanceDescription(t *testing.T) {
	enhancer := NewLLMQueryEnhancer()
	
	tests := []struct {
		name        string
		metricInfo  *MetricInfo
		suggestion  QuerySuggestion
		expectContains string
	}{
		{
			name: "HTTP metric enhancement",
			metricInfo: &MetricInfo{
				Name: "http_requests_total",
				Type: MetricTypeCounter,
			},
			suggestion: QuerySuggestion{
				Query:       "rate(http_requests_total[5m])",
				Description: "Rate per second over 5 minutes",
			},
			expectContains: "HTTP",
		},
		{
			name: "Error metric enhancement",
			metricInfo: &MetricInfo{
				Name: "error_count",
				Type: MetricTypeCounter,
			},
			suggestion: QuerySuggestion{
				Query:       "rate(error_count[5m])",
				Description: "Rate per second over 5 minutes",
			},
			expectContains: "Error",
		},
		{
			name: "Memory metric enhancement",
			metricInfo: &MetricInfo{
				Name: "memory_usage",
				Type: MetricTypeGauge,
			},
			suggestion: QuerySuggestion{
				Description: "Current value",
			},
			expectContains: "Resource Usage",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enhanced := enhancer.enhanceDescription(tt.metricInfo, tt.suggestion)
			
			if enhanced == "" {
				t.Error("Enhanced description should not be empty")
			}
			
			// For these specific tests, we expect some enhancement
			if enhanced == tt.suggestion.Description && tt.expectContains != "" {
				t.Errorf("Expected description to be enhanced with %s, but got original: %s", tt.expectContains, enhanced)
			}
		})
	}
}

func TestOptimizeQuery(t *testing.T) {
	enhancer := NewLLMQueryEnhancer()
	
	tests := []struct {
		name       string
		metricInfo *MetricInfo
		query      string
		expectDiff bool // Whether we expect the query to be different
	}{
		{
			name: "High-frequency HTTP metric optimization",
			metricInfo: &MetricInfo{
				Name: "http_requests_total",
				Type: MetricTypeCounter,
			},
			query:      "rate(http_requests_total[5m])",
			expectDiff: true, // Should optimize to 2m interval
		},
		{
			name: "Histogram query optimization",
			metricInfo: &MetricInfo{
				Name: "http_duration_bucket",
				Type: MetricTypeHistogram,
			},
			query:      "histogram_quantile(0.95, rate(http_duration_bucket[5m]))",
			expectDiff: true, // Should add proper aggregation
		},
		{
			name: "Simple gauge query",
			metricInfo: &MetricInfo{
				Name: "memory_usage",
				Type: MetricTypeGauge,
			},
			query:      "memory_usage",
			expectDiff: false, // Should remain unchanged
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			optimized := enhancer.optimizeQuery(tt.metricInfo, tt.query)
			
			if optimized == "" {
				t.Error("Optimized query should not be empty")
			}
			
			if tt.expectDiff && optimized == tt.query {
				t.Errorf("Expected query to be optimized, but got same: %s", optimized)
			}
			
			if !tt.expectDiff && optimized != tt.query {
				t.Errorf("Expected query to remain unchanged, but got: %s (original: %s)", optimized, tt.query)
			}
		})
	}
}

func TestSuggestVisualizationType(t *testing.T) {
	enhancer := NewLLMQueryEnhancer()
	
	tests := []struct {
		name       string
		metricInfo *MetricInfo
		suggestion QuerySuggestion
		expected   string
	}{
		{
			name: "Histogram quantile should be timeseries",
			metricInfo: &MetricInfo{
				Name: "http_duration_bucket",
				Type: MetricTypeHistogram,
			},
			suggestion: QuerySuggestion{
				Query:             "histogram_quantile(0.95, rate(http_duration_bucket[5m]))",
				VisualizationType: "stat",
			},
			expected: "timeseries",
		},
		{
			name: "Average should be stat",
			suggestion: QuerySuggestion{
				Query:             "avg(memory_usage)",
				VisualizationType: "timeseries",
			},
			expected: "stat",
		},
		{
			name: "Percentage metric should be gauge",
			metricInfo: &MetricInfo{
				Name: "cpu_percent",
				Type: MetricTypeGauge,
			},
			suggestion: QuerySuggestion{
				VisualizationType: "timeseries",
			},
			expected: "gauge",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := enhancer.suggestVisualizationType(tt.metricInfo, tt.suggestion)
			
			if result != tt.expected {
				t.Errorf("Expected visualization type %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGenerateContextualQueries(t *testing.T) {
	enhancer := NewLLMQueryEnhancer()
	
	tests := []struct {
		name       string
		metricInfo *MetricInfo
		expectMin  int // Minimum number of contextual queries expected
	}{
		{
			name: "HTTP request metric",
			metricInfo: &MetricInfo{
				Name: "http_requests_total",
				Type: MetricTypeCounter,
			},
			expectMin: 1, // Should generate error rate and success rate queries
		},
		{
			name: "Error metric",
			metricInfo: &MetricInfo{
				Name: "error_count",
				Type: MetricTypeCounter,
			},
			expectMin: 1, // Should generate alert query
		},
		{
			name: "CPU metric",
			metricInfo: &MetricInfo{
				Name: "cpu_usage",
				Type: MetricTypeGauge,
			},
			expectMin: 1, // Should generate high usage alert
		},
		{
			name: "Memory metric",
			metricInfo: &MetricInfo{
				Name: "memory_bytes",
				Type: MetricTypeGauge,
			},
			expectMin: 1, // Should generate GB conversion
		},
		{
			name: "Generic metric",
			metricInfo: &MetricInfo{
				Name: "generic_metric",
				Type: MetricTypeGauge,
			},
			expectMin: 0, // No specific contextual queries
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contextual := enhancer.generateContextualQueries(tt.metricInfo)
			
			if len(contextual) < tt.expectMin {
				t.Errorf("Expected at least %d contextual queries, got %d", tt.expectMin, len(contextual))
			}
			
			// Verify query structure
			for _, query := range contextual {
				if query.Query == "" {
					t.Error("Contextual query should not be empty")
				}
				if query.Description == "" {
					t.Error("Contextual query description should not be empty")
				}
				if query.VisualizationType == "" {
					t.Error("Contextual query visualization type should not be empty")
				}
			}
		})
	}
}

func TestExtractPercentile(t *testing.T) {
	tests := []struct {
		query    string
		expected string
	}{
		{
			query:    "histogram_quantile(0.50, rate(metric[5m]))",
			expected: "50th",
		},
		{
			query:    "histogram_quantile(0.95, rate(metric[5m]))",
			expected: "95th",
		},
		{
			query:    "histogram_quantile(0.99, rate(metric[5m]))",
			expected: "99th",
		},
		{
			query:    "histogram_quantile(0.90, rate(metric[5m]))",
			expected: "90th",
		},
		{
			query:    "rate(metric[5m])",
			expected: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			result := extractPercentile(tt.query)
			if result != tt.expected {
				t.Errorf("extractPercentile(%s) = %s, want %s", tt.query, result, tt.expected)
			}
		})
	}
}

func TestExtractMetricNameFromHistogramQuery(t *testing.T) {
	tests := []struct {
		query    string
		expected string
	}{
		{
			query:    "histogram_quantile(0.95, rate(http_duration_bucket[5m]))",
			expected: "http_duration",
		},
		{
			query:    "rate(request_duration_bucket[5m])",
			expected: "request_duration",
		},
		{
			query:    "sum(rate(api_latency_bucket[2m])) by (le)",
			expected: "api_latency",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			result := extractMetricNameFromHistogramQuery(tt.query)
			if result != tt.expected {
				t.Errorf("extractMetricNameFromHistogramQuery(%s) = %s, want %s", tt.query, result, tt.expected)
			}
		})
	}
}

// Benchmark tests for performance verification
func BenchmarkEnhanceQueries(b *testing.B) {
	enhancer := NewLLMQueryEnhancer()
	metricInfo := &MetricInfo{
		Name: "http_requests_total",
		Type: MetricTypeCounter,
	}
	suggestions := []QuerySuggestion{
		{
			Query:             "rate(http_requests_total[5m])",
			Description:       "Rate per second over 5 minutes",
			VisualizationType: "timeseries",
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enhancer.EnhanceQueries(context.Background(), metricInfo, suggestions)
	}
}

func BenchmarkGenerateContextualQueries(b *testing.B) {
	enhancer := NewLLMQueryEnhancer()
	metricInfo := &MetricInfo{
		Name: "http_requests_total",
		Type: MetricTypeCounter,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enhancer.generateContextualQueries(metricInfo)
	}
}