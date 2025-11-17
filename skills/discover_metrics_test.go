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

func TestNewDiscoverMetricsSkill(t *testing.T) {
	logger := zap.NewNop()
	fakePromQL := &promqlfakes.FakePromQL{}

	skill := NewDiscoverMetricsSkill(logger, fakePromQL)

	if skill == nil {
		t.Error("Expected non-nil skill")
	}
}

func TestDiscoverMetricsHandler(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name          string
		args          map[string]any
		setupMock     func(*promqlfakes.FakePromQL)
		wantErr       bool
		expectedError string
		validateFunc  func(t *testing.T, result string)
	}{
		{
			name: "successful discovery without filters",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
			},
			setupMock: func(fake *promqlfakes.FakePromQL) {
				fake.DiscoverMetricsReturns([]promql.MetricInfo{
					{
						Name:   "http_requests_total",
						Type:   promql.MetricTypeCounter,
						Help:   "Total HTTP requests",
						Labels: []string{"method", "status"},
					},
					{
						Name:   "process_cpu_usage",
						Type:   promql.MetricTypeGauge,
						Help:   "CPU usage",
						Labels: []string{"instance"},
					},
				}, nil)
			},
			wantErr: false,
			validateFunc: func(t *testing.T, result string) {
				var response DiscoverMetricsResponse
				if err := json.Unmarshal([]byte(result), &response); err != nil {
					t.Fatalf("Expected valid JSON result, got error: %v", err)
				}
				if response.PrometheusURL != "http://prometheus.test:9090" {
					t.Errorf("Expected prometheus_url 'http://prometheus.test:9090', got %s", response.PrometheusURL)
				}
				if response.TotalMetrics != 2 {
					t.Errorf("Expected 2 total metrics, got %d", response.TotalMetrics)
				}
				if len(response.Metrics) != 2 {
					t.Errorf("Expected 2 metrics, got %d", len(response.Metrics))
				}
			},
		},
		{
			name: "successful discovery with name pattern",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"name_pattern":   "^http_.*",
			},
			setupMock: func(fake *promqlfakes.FakePromQL) {
				fake.DiscoverMetricsReturns([]promql.MetricInfo{
					{
						Name:   "http_requests_total",
						Type:   promql.MetricTypeCounter,
						Help:   "Total HTTP requests",
						Labels: []string{"method"},
					},
				}, nil)
			},
			wantErr: false,
			validateFunc: func(t *testing.T, result string) {
				var response DiscoverMetricsResponse
				if err := json.Unmarshal([]byte(result), &response); err != nil {
					t.Fatalf("Expected valid JSON result, got error: %v", err)
				}
				if response.Filters.NamePattern != "^http_.*" {
					t.Errorf("Expected name_pattern '^http_.*', got %s", response.Filters.NamePattern)
				}
				if response.TotalMetrics != 1 {
					t.Errorf("Expected 1 total metric, got %d", response.TotalMetrics)
				}
			},
		},
		{
			name: "successful discovery with metric type filter",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"metric_type":    "counter",
			},
			setupMock: func(fake *promqlfakes.FakePromQL) {
				fake.DiscoverMetricsReturns([]promql.MetricInfo{
					{
						Name:   "http_requests_total",
						Type:   promql.MetricTypeCounter,
						Help:   "Total HTTP requests",
						Labels: []string{"method"},
					},
				}, nil)
			},
			wantErr: false,
			validateFunc: func(t *testing.T, result string) {
				var response DiscoverMetricsResponse
				if err := json.Unmarshal([]byte(result), &response); err != nil {
					t.Fatalf("Expected valid JSON result, got error: %v", err)
				}
				if response.Filters.MetricType != "counter" {
					t.Errorf("Expected metric_type 'counter', got %s", response.Filters.MetricType)
				}
				if len(response.Metrics) != 1 {
					t.Errorf("Expected 1 metric, got %d", len(response.Metrics))
				}
				if response.Metrics[0].Type != promql.MetricTypeCounter {
					t.Errorf("Expected counter type, got %s", response.Metrics[0].Type)
				}
			},
		},
		{
			name: "successful discovery with both filters",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"name_pattern":   "http_.*",
				"metric_type":    "gauge",
			},
			setupMock: func(fake *promqlfakes.FakePromQL) {
				fake.DiscoverMetricsReturns([]promql.MetricInfo{
					{
						Name:   "http_response_size_bytes",
						Type:   promql.MetricTypeGauge,
						Help:   "HTTP response size",
						Labels: []string{"method"},
					},
				}, nil)
			},
			wantErr: false,
			validateFunc: func(t *testing.T, result string) {
				var response DiscoverMetricsResponse
				if err := json.Unmarshal([]byte(result), &response); err != nil {
					t.Fatalf("Expected valid JSON result, got error: %v", err)
				}
				if response.Filters.NamePattern != "http_.*" {
					t.Errorf("Expected name_pattern 'http_.*', got %s", response.Filters.NamePattern)
				}
				if response.Filters.MetricType != "gauge" {
					t.Errorf("Expected metric_type 'gauge', got %s", response.Filters.MetricType)
				}
			},
		},
		{
			name: "missing prometheus_url",
			args: map[string]any{
				"name_pattern": ".*",
			},
			setupMock:     func(fake *promqlfakes.FakePromQL) {},
			wantErr:       true,
			expectedError: "prometheus_url is required and must be a string",
		},
		{
			name: "empty prometheus_url",
			args: map[string]any{
				"prometheus_url": "",
			},
			setupMock:     func(fake *promqlfakes.FakePromQL) {},
			wantErr:       true,
			expectedError: "prometheus_url is required and must be a string",
		},
		{
			name: "prometheus connection error",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
			},
			setupMock: func(fake *promqlfakes.FakePromQL) {
				fake.DiscoverMetricsReturns(nil, errors.New("connection refused"))
			},
			wantErr:       true,
			expectedError: "failed to discover metrics: connection refused",
		},
		{
			name: "no metrics found",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"name_pattern":   "non_existent_.*",
			},
			setupMock: func(fake *promqlfakes.FakePromQL) {
				fake.DiscoverMetricsReturns([]promql.MetricInfo{}, nil)
			},
			wantErr: false,
			validateFunc: func(t *testing.T, result string) {
				var response DiscoverMetricsResponse
				if err := json.Unmarshal([]byte(result), &response); err != nil {
					t.Fatalf("Expected valid JSON result, got error: %v", err)
				}
				if response.TotalMetrics != 0 {
					t.Errorf("Expected 0 total metrics, got %d", response.TotalMetrics)
				}
				if len(response.Metrics) != 0 {
					t.Errorf("Expected empty metrics array, got %d", len(response.Metrics))
				}
			},
		},
		{
			name: "all metric types",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
			},
			setupMock: func(fake *promqlfakes.FakePromQL) {
				fake.DiscoverMetricsReturns([]promql.MetricInfo{
					{
						Name:   "requests_total",
						Type:   promql.MetricTypeCounter,
						Help:   "Counter metric",
						Labels: []string{"instance"},
					},
					{
						Name:   "memory_usage",
						Type:   promql.MetricTypeGauge,
						Help:   "Gauge metric",
						Labels: []string{"instance"},
					},
					{
						Name:   "request_duration_seconds",
						Type:   promql.MetricTypeHistogram,
						Help:   "Histogram metric",
						Labels: []string{"le"},
					},
					{
						Name:   "response_size",
						Type:   promql.MetricTypeSummary,
						Help:   "Summary metric",
						Labels: []string{"quantile"},
					},
				}, nil)
			},
			wantErr: false,
			validateFunc: func(t *testing.T, result string) {
				var response DiscoverMetricsResponse
				if err := json.Unmarshal([]byte(result), &response); err != nil {
					t.Fatalf("Expected valid JSON result, got error: %v", err)
				}
				if response.TotalMetrics != 4 {
					t.Errorf("Expected 4 total metrics, got %d", response.TotalMetrics)
				}
				expectedTypes := map[string]promql.MetricType{
					"requests_total":           promql.MetricTypeCounter,
					"memory_usage":             promql.MetricTypeGauge,
					"request_duration_seconds": promql.MetricTypeHistogram,
					"response_size":            promql.MetricTypeSummary,
				}
				for _, metric := range response.Metrics {
					if expectedType, ok := expectedTypes[metric.Name]; ok {
						if metric.Type != expectedType {
							t.Errorf("Expected type %s for %s, got %s", expectedType, metric.Name, metric.Type)
						}
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakePromQL := &promqlfakes.FakePromQL{}
			tt.setupMock(fakePromQL)

			skill := &DiscoverMetricsSkill{
				logger: logger,
				promql: fakePromQL,
			}

			result, err := skill.DiscoverMetricsHandler(context.Background(), tt.args)

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
