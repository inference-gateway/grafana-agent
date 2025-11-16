package skills

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/inference-gateway/grafana-agent/internal/promql"
	"go.uber.org/zap"
)

// mockPromQLServiceForGenerate is a mock implementation for testing generate_promql_queries
type mockPromQLServiceForGenerate struct {
	getMetricMetadataFunc func(ctx context.Context, prometheusURL, metricName string) (*promql.MetricInfo, error)
	generateQueriesFunc   func(metricInfo *promql.MetricInfo) []promql.QuerySuggestion
}

func (m *mockPromQLServiceForGenerate) GetMetricMetadata(ctx context.Context, prometheusURL, metricName string) (*promql.MetricInfo, error) {
	if m.getMetricMetadataFunc != nil {
		return m.getMetricMetadataFunc(ctx, prometheusURL, metricName)
	}
	return &promql.MetricInfo{
		Name:   metricName,
		Type:   promql.MetricTypeCounter,
		Help:   "Test metric",
		Labels: []string{"instance", "job"},
	}, nil
}

func (m *mockPromQLServiceForGenerate) GenerateQueries(metricInfo *promql.MetricInfo) []promql.QuerySuggestion {
	if m.generateQueriesFunc != nil {
		return m.generateQueriesFunc(metricInfo)
	}
	return []promql.QuerySuggestion{
		{
			Query:             "rate(" + metricInfo.Name + "[5m])",
			Description:       "Rate of change",
			VisualizationType: "timeseries",
			YAxisLabel:        "rate",
		},
	}
}

func (m *mockPromQLServiceForGenerate) ValidateQuery(ctx context.Context, prometheusURL, query string) error {
	return nil
}

func (m *mockPromQLServiceForGenerate) GetBestQuery(suggestions []promql.QuerySuggestion) promql.QuerySuggestion {
	if len(suggestions) > 0 {
		return suggestions[0]
	}
	return promql.QuerySuggestion{}
}

func TestNewGeneratePromqlQueriesSkill(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockPromQL := &mockPromQLServiceForGenerate{}

	skill := NewGeneratePromqlQueriesSkill(logger, mockPromQL)

	if skill == nil {
		t.Error("Expected non-nil skill")
	}
}

func TestGeneratePromqlQueriesHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name          string
		args          map[string]any
		mockPromQL    *mockPromQLServiceForGenerate
		wantErr       bool
		expectedError string
		validateFunc  func(t *testing.T, result string)
	}{
		{
			name: "successful query generation",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"metric_names":   []any{"http_requests_total", "http_duration_seconds"},
			},
			mockPromQL: &mockPromQLServiceForGenerate{},
			wantErr:    false,
			validateFunc: func(t *testing.T, result string) {
				var response GeneratePromqlQueriesResponse
				if err := json.Unmarshal([]byte(result), &response); err != nil {
					t.Fatalf("Expected valid JSON result, got error: %v", err)
				}
				if response.PrometheusURL != "http://prometheus.test:9090" {
					t.Errorf("Expected prometheus_url 'http://prometheus.test:9090', got %s", response.PrometheusURL)
				}
				if len(response.Results) != 2 {
					t.Errorf("Expected 2 results, got %d", len(response.Results))
				}
				for _, result := range response.Results {
					if result.MetricName == "" {
						t.Error("Expected non-empty metric name")
					}
					if len(result.Suggestions) == 0 {
						t.Errorf("Expected suggestions for metric %s", result.MetricName)
					}
				}
			},
		},
		{
			name: "missing prometheus_url",
			args: map[string]any{
				"metric_names": []any{"http_requests_total"},
			},
			mockPromQL:    &mockPromQLServiceForGenerate{},
			wantErr:       true,
			expectedError: "prometheus_url is required and must be a string",
		},
		{
			name: "missing metric_names",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
			},
			mockPromQL:    &mockPromQLServiceForGenerate{},
			wantErr:       true,
			expectedError: "metric_names is required",
		},
		{
			name: "empty metric_names array",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"metric_names":   []any{},
			},
			mockPromQL:    &mockPromQLServiceForGenerate{},
			wantErr:       true,
			expectedError: "metric_names cannot be empty",
		},
		{
			name: "invalid metric_names type",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"metric_names":   "not_an_array",
			},
			mockPromQL:    &mockPromQLServiceForGenerate{},
			wantErr:       true,
			expectedError: "metric_names must be an array",
		},
		{
			name: "metadata fetch error",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"metric_names":   []any{"http_requests_total"},
			},
			mockPromQL: &mockPromQLServiceForGenerate{
				getMetricMetadataFunc: func(ctx context.Context, prometheusURL, metricName string) (*promql.MetricInfo, error) {
					return nil, errors.New("prometheus connection error")
				},
			},
			wantErr: false,
			validateFunc: func(t *testing.T, result string) {
				var response GeneratePromqlQueriesResponse
				if err := json.Unmarshal([]byte(result), &response); err != nil {
					t.Fatalf("Expected valid JSON result, got error: %v", err)
				}
				if len(response.Results) != 1 {
					t.Errorf("Expected 1 result, got %d", len(response.Results))
				}
				if response.Results[0].Error == "" {
					t.Error("Expected error in result for failed metadata fetch")
				}
			},
		},
		{
			name: "no query suggestions generated",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"metric_names":   []any{"unknown_metric"},
			},
			mockPromQL: &mockPromQLServiceForGenerate{
				generateQueriesFunc: func(metricInfo *promql.MetricInfo) []promql.QuerySuggestion {
					return []promql.QuerySuggestion{}
				},
			},
			wantErr: false,
			validateFunc: func(t *testing.T, result string) {
				var response GeneratePromqlQueriesResponse
				if err := json.Unmarshal([]byte(result), &response); err != nil {
					t.Fatalf("Expected valid JSON result, got error: %v", err)
				}
				if len(response.Results) != 1 {
					t.Errorf("Expected 1 result, got %d", len(response.Results))
				}
				if response.Results[0].Error == "" {
					t.Error("Expected error in result for no suggestions")
				}
				expectedError := "no query suggestions could be generated"
				if response.Results[0].Error != expectedError {
					t.Errorf("Expected error '%s', got '%s'", expectedError, response.Results[0].Error)
				}
			},
		},
		{
			name: "multiple metrics with different types",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"metric_names":   []any{"counter_metric", "gauge_metric", "histogram_metric"},
			},
			mockPromQL: &mockPromQLServiceForGenerate{
				getMetricMetadataFunc: func(ctx context.Context, prometheusURL, metricName string) (*promql.MetricInfo, error) {
					typeMap := map[string]promql.MetricType{
						"counter_metric":   promql.MetricTypeCounter,
						"gauge_metric":     promql.MetricTypeGauge,
						"histogram_metric": promql.MetricTypeHistogram,
					}
					return &promql.MetricInfo{
						Name:   metricName,
						Type:   typeMap[metricName],
						Help:   "Test metric " + metricName,
						Labels: []string{"instance"},
					}, nil
				},
			},
			wantErr: false,
			validateFunc: func(t *testing.T, result string) {
				var response GeneratePromqlQueriesResponse
				if err := json.Unmarshal([]byte(result), &response); err != nil {
					t.Fatalf("Expected valid JSON result, got error: %v", err)
				}
				if len(response.Results) != 3 {
					t.Errorf("Expected 3 results, got %d", len(response.Results))
				}
				expectedTypes := map[string]string{
					"counter_metric":   "counter",
					"gauge_metric":     "gauge",
					"histogram_metric": "histogram",
				}
				for _, result := range response.Results {
					if expectedType, ok := expectedTypes[result.MetricName]; ok {
						if result.MetricType != expectedType {
							t.Errorf("Expected metric type '%s' for %s, got '%s'", expectedType, result.MetricName, result.MetricType)
						}
					}
					if len(result.Suggestions) == 0 {
						t.Errorf("Expected suggestions for metric %s", result.MetricName)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill := &GeneratePromqlQueriesSkill{
				logger: logger,
				promql: tt.mockPromQL,
			}

			result, err := skill.GeneratePromqlQueriesHandler(context.Background(), tt.args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.expectedError != "" && err.Error() != tt.expectedError {
					t.Errorf("Expected error '%s', got '%s'", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}
