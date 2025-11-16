package skills

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/inference-gateway/grafana-agent/internal/promql"
	"github.com/inference-gateway/grafana-agent/internal/promql/promqlfakes"
	"go.uber.org/zap"
)

func TestNewGeneratePromqlQueriesSkill(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	fakePromQL := &promqlfakes.FakePromQL{}

	skill := NewGeneratePromqlQueriesSkill(logger, fakePromQL)

	if skill == nil {
		t.Error("Expected non-nil skill")
	}
}

func TestGeneratePromqlQueriesHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name          string
		args          map[string]any
		setupMock     func(*promqlfakes.FakePromQL)
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
			setupMock: func(fake *promqlfakes.FakePromQL) {
				fake.GetMetricMetadataReturns(&promql.MetricInfo{
					Name:   "test_metric",
					Type:   promql.MetricTypeCounter,
					Help:   "Test metric",
					Labels: []string{"instance", "job"},
				}, nil)
				fake.GenerateQueriesReturns([]promql.QuerySuggestion{
					{
						Query:             "rate(test_metric[5m])",
						Description:       "Rate of change",
						VisualizationType: "timeseries",
						YAxisLabel:        "rate",
					},
				})
			},
			wantErr: false,
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
			setupMock:     func(fake *promqlfakes.FakePromQL) {},
			wantErr:       true,
			expectedError: "prometheus_url is required and must be a string",
		},
		{
			name: "missing metric_names",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
			},
			setupMock:     func(fake *promqlfakes.FakePromQL) {},
			wantErr:       true,
			expectedError: "metric_names is required",
		},
		{
			name: "empty metric_names array",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"metric_names":   []any{},
			},
			setupMock:     func(fake *promqlfakes.FakePromQL) {},
			wantErr:       true,
			expectedError: "metric_names cannot be empty",
		},
		{
			name: "invalid metric_names type",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"metric_names":   "not_an_array",
			},
			setupMock:     func(fake *promqlfakes.FakePromQL) {},
			wantErr:       true,
			expectedError: "metric_names must be an array",
		},
		{
			name: "metadata fetch error",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"metric_names":   []any{"http_requests_total"},
			},
			setupMock: func(fake *promqlfakes.FakePromQL) {
				fake.GetMetricMetadataReturns(nil, errors.New("prometheus connection error"))
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
			setupMock: func(fake *promqlfakes.FakePromQL) {
				fake.GetMetricMetadataReturns(&promql.MetricInfo{
					Name: "unknown_metric",
					Type: promql.MetricTypeUnknown,
					Help: "Unknown metric",
				}, nil)
				fake.GenerateQueriesReturns([]promql.QuerySuggestion{})
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
			setupMock: func(fake *promqlfakes.FakePromQL) {
				fake.GetMetricMetadataStub = func(ctx context.Context, prometheusURL, metricName string) (*promql.MetricInfo, error) {
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
				}
				fake.GenerateQueriesReturns([]promql.QuerySuggestion{
					{
						Query:             "test_query",
						Description:       "Test description",
						VisualizationType: "timeseries",
						YAxisLabel:        "value",
					},
				})
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
			fakePromQL := &promqlfakes.FakePromQL{}
			tt.setupMock(fakePromQL)

			skill := &GeneratePromqlQueriesSkill{
				logger: logger,
				promql: fakePromQL,
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
