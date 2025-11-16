package skills

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/inference-gateway/grafana-agent/internal/promql/promqlfakes"
	"go.uber.org/zap"
)

func TestNewValidatePromqlQuerySkill(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	fakePromQL := &promqlfakes.FakePromQL{}

	skill := NewValidatePromqlQuerySkill(logger, fakePromQL)

	if skill == nil {
		t.Error("Expected non-nil skill")
	}
}

func TestValidatePromqlQueryHandler(t *testing.T) {
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
			name: "valid query",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"query":          "up",
			},
			setupMock: func(fake *promqlfakes.FakePromQL) {
				fake.ValidateQueryReturns(nil)
			},
			wantErr: false,
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
			setupMock: func(fake *promqlfakes.FakePromQL) {
				fake.ValidateQueryReturns(errors.New("parse error: unexpected left brace"))
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
			setupMock:     func(fake *promqlfakes.FakePromQL) {},
			wantErr:       true,
			expectedError: "prometheus_url is required and must be a string",
		},
		{
			name: "empty prometheus_url",
			args: map[string]any{
				"prometheus_url": "",
				"query":          "up",
			},
			setupMock:     func(fake *promqlfakes.FakePromQL) {},
			wantErr:       true,
			expectedError: "prometheus_url is required and must be a string",
		},
		{
			name: "missing query",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
			},
			setupMock:     func(fake *promqlfakes.FakePromQL) {},
			wantErr:       true,
			expectedError: "query is required and must be a string",
		},
		{
			name: "empty query",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"query":          "",
			},
			setupMock:     func(fake *promqlfakes.FakePromQL) {},
			wantErr:       true,
			expectedError: "query is required and must be a string",
		},
		{
			name: "complex valid query",
			args: map[string]any{
				"prometheus_url": "http://prometheus.test:9090",
				"query":          "rate(http_requests_total{status=\"200\"}[5m])",
			},
			setupMock: func(fake *promqlfakes.FakePromQL) {
				fake.ValidateQueryReturns(nil)
			},
			wantErr: false,
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
			setupMock: func(fake *promqlfakes.FakePromQL) {
				fake.ValidateQueryReturns(nil)
			},
			wantErr: false,
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
			setupMock: func(fake *promqlfakes.FakePromQL) {
				fake.ValidateQueryReturns(errors.New("connection refused"))
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
			setupMock: func(fake *promqlfakes.FakePromQL) {
				fake.ValidateQueryReturns(nil)
			},
			wantErr: false,
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
			fakePromQL := &promqlfakes.FakePromQL{}
			tt.setupMock(fakePromQL)

			skill := &ValidatePromqlQuerySkill{
				logger: logger,
				promql: fakePromQL,
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
