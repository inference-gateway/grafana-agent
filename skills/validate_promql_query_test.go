package skills

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/inference-gateway/grafana-agent/internal/promql"
	"go.uber.org/zap"
)

// mockPromQLServiceForValidate is a mock implementation for testing validate_promql_query
type mockPromQLServiceForValidate struct {
	validateQueryFunc func(ctx context.Context, prometheusURL, query string) error
}

func (m *mockPromQLServiceForValidate) GetMetricMetadata(ctx context.Context, prometheusURL, metricName string) (*promql.MetricInfo, error) {
	return nil, nil
}

func (m *mockPromQLServiceForValidate) GenerateQueries(metricInfo *promql.MetricInfo) []promql.QuerySuggestion {
	return nil
}

func (m *mockPromQLServiceForValidate) ValidateQuery(ctx context.Context, prometheusURL, query string) error {
	if m.validateQueryFunc != nil {
		return m.validateQueryFunc(ctx, prometheusURL, query)
	}
	return nil
}

func (m *mockPromQLServiceForValidate) GetBestQuery(suggestions []promql.QuerySuggestion) promql.QuerySuggestion {
	return promql.QuerySuggestion{}
}

func TestNewValidatePromqlQuerySkill(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockPromQL := &mockPromQLServiceForValidate{}

	skill := NewValidatePromqlQuerySkill(logger, mockPromQL)

	if skill == nil {
		t.Error("Expected non-nil skill")
	}
}

func TestValidatePromqlQueryHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name          string
		args          map[string]any
		mockPromQL    *mockPromQLServiceForValidate
		wantErr       bool
		expectedError string
		validateFunc  func(t *testing.T, result string)
	}{
		{
			name: "valid query",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"query":          "up",
			},
			mockPromQL: &mockPromQLServiceForValidate{},
			wantErr:    false,
			validateFunc: func(t *testing.T, result string) {
				var response ValidateQueryResponse
				if err := json.Unmarshal([]byte(result), &response); err != nil {
					t.Fatalf("Expected valid JSON result, got error: %v", err)
				}
				if response.PrometheusURL != "http://prometheus.test:9090" {
					t.Errorf("Expected prometheus_url 'http://prometheus.test:9090', got %s", response.PrometheusURL)
				}
				if response.Query != "up" {
					t.Errorf("Expected query 'up', got %s", response.Query)
				}
				if !response.Valid {
					t.Error("Expected valid query")
				}
				if response.Error != "" {
					t.Errorf("Expected no error, got %s", response.Error)
				}
			},
		},
		{
			name: "invalid query",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"query":          "invalid{syntax",
			},
			mockPromQL: &mockPromQLServiceForValidate{
				validateQueryFunc: func(ctx context.Context, prometheusURL, query string) error {
					return errors.New("parse error: unexpected left brace")
				},
			},
			wantErr: false,
			validateFunc: func(t *testing.T, result string) {
				var response ValidateQueryResponse
				if err := json.Unmarshal([]byte(result), &response); err != nil {
					t.Fatalf("Expected valid JSON result, got error: %v", err)
				}
				if response.Valid {
					t.Error("Expected invalid query")
				}
				if response.Error == "" {
					t.Error("Expected error message")
				}
				if response.Error != "parse error: unexpected left brace" {
					t.Errorf("Expected specific error, got %s", response.Error)
				}
			},
		},
		{
			name: "missing prometheus_url",
			args: map[string]any{
				"query": "up",
			},
			mockPromQL:    &mockPromQLServiceForValidate{},
			wantErr:       true,
			expectedError: "prometheus_url is required and must be a string",
		},
		{
			name: "empty prometheus_url",
			args: map[string]any{
				"prometheus_url": "",
				"query":          "up",
			},
			mockPromQL:    &mockPromQLServiceForValidate{},
			wantErr:       true,
			expectedError: "prometheus_url is required and must be a string",
		},
		{
			name: "missing query",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
			},
			mockPromQL:    &mockPromQLServiceForValidate{},
			wantErr:       true,
			expectedError: "query is required and must be a string",
		},
		{
			name: "empty query",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"query":          "",
			},
			mockPromQL:    &mockPromQLServiceForValidate{},
			wantErr:       true,
			expectedError: "query is required and must be a string",
		},
		{
			name: "complex valid query",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"query":          "rate(http_requests_total{status=\"200\"}[5m])",
			},
			mockPromQL: &mockPromQLServiceForValidate{},
			wantErr:    false,
			validateFunc: func(t *testing.T, result string) {
				var response ValidateQueryResponse
				if err := json.Unmarshal([]byte(result), &response); err != nil {
					t.Fatalf("Expected valid JSON result, got error: %v", err)
				}
				if !response.Valid {
					t.Errorf("Expected valid query, got error: %s", response.Error)
				}
			},
		},
		{
			name: "histogram query validation",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"query":          "histogram_quantile(0.95, rate(http_duration_bucket[5m]))",
			},
			mockPromQL: &mockPromQLServiceForValidate{},
			wantErr:    false,
			validateFunc: func(t *testing.T, result string) {
				var response ValidateQueryResponse
				if err := json.Unmarshal([]byte(result), &response); err != nil {
					t.Fatalf("Expected valid JSON result, got error: %v", err)
				}
				if !response.Valid {
					t.Errorf("Expected valid query, got error: %s", response.Error)
				}
			},
		},
		{
			name: "prometheus connection error",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"query":          "up",
			},
			mockPromQL: &mockPromQLServiceForValidate{
				validateQueryFunc: func(ctx context.Context, prometheusURL, query string) error {
					return errors.New("connection refused")
				},
			},
			wantErr: false,
			validateFunc: func(t *testing.T, result string) {
				var response ValidateQueryResponse
				if err := json.Unmarshal([]byte(result), &response); err != nil {
					t.Fatalf("Expected valid JSON result, got error: %v", err)
				}
				if response.Valid {
					t.Error("Expected invalid due to connection error")
				}
				if response.Error != "connection refused" {
					t.Errorf("Expected connection error, got %s", response.Error)
				}
			},
		},
		{
			name: "query with aggregation",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"query":          "sum by (instance) (rate(cpu_usage[5m]))",
			},
			mockPromQL: &mockPromQLServiceForValidate{},
			wantErr:    false,
			validateFunc: func(t *testing.T, result string) {
				var response ValidateQueryResponse
				if err := json.Unmarshal([]byte(result), &response); err != nil {
					t.Fatalf("Expected valid JSON result, got error: %v", err)
				}
				if !response.Valid {
					t.Errorf("Expected valid query, got error: %s", response.Error)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill := &ValidatePromqlQuerySkill{
				logger: logger,
				promql: tt.mockPromQL,
			}

			result, err := skill.ValidatePromqlQueryHandler(context.Background(), tt.args)

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
