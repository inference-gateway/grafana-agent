package skills

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/inference-gateway/grafana-agent/config"
	"github.com/inference-gateway/grafana-agent/internal/grafana"
	"go.uber.org/zap"
)

// mockGrafanaService is a mock implementation of the Grafana interface for testing
type mockGrafanaService struct {
	createDashboardFunc func(ctx context.Context, dashboard grafana.Dashboard, grafanaURL, apiKey string) (*grafana.DashboardResponse, error)
}

func (m *mockGrafanaService) CreateDashboard(ctx context.Context, dashboard grafana.Dashboard, grafanaURL, apiKey string) (*grafana.DashboardResponse, error) {
	if m.createDashboardFunc != nil {
		return m.createDashboardFunc(ctx, dashboard, grafanaURL, apiKey)
	}
	return &grafana.DashboardResponse{
		ID:  123,
		UID: "test-uid",
		URL: "/d/test-uid/test-dashboard",
	}, nil
}

func (m *mockGrafanaService) UpdateDashboard(ctx context.Context, dashboard grafana.Dashboard, grafanaURL, apiKey string) (*grafana.DashboardResponse, error) {
	return m.CreateDashboard(ctx, dashboard, grafanaURL, apiKey)
}

func (m *mockGrafanaService) GetDashboard(ctx context.Context, uid, grafanaURL, apiKey string) (*grafana.Dashboard, error) {
	return nil, nil
}

func (m *mockGrafanaService) DeleteDashboard(ctx context.Context, uid, grafanaURL, apiKey string) error {
	return nil
}

func TestNewCreateDashboardSkill(t *testing.T) {
	logger := zap.NewNop()
	mockGrafana := &mockGrafanaService{}
	config := &config.GrafanaConfig{
		DeployEnabled: true,
		URL:           "http://grafana.test",
		APIKey:        "test-key",
	}

	skill := NewCreateDashboardSkill(logger, mockGrafana, config)

	if skill == nil {
		t.Error("Expected non-nil skill")
	}
}

func TestCreateDashboardHandler_BasicPanels(t *testing.T) {
	logger := zap.NewNop()
	mockGrafana := &mockGrafanaService{}
	config := &config.GrafanaConfig{
		DeployEnabled: false,
	}

	skill := &CreateDashboardSkill{
		logger:  logger,
		grafana: mockGrafana,
		config:  config,
	}

	args := map[string]any{
		"dashboard_title": "Test Dashboard",
		"panels": []any{
			map[string]any{
				"title": "Test Panel",
				"type":  "timeseries",
				"targets": []any{
					map[string]any{
						"refId": "A",
						"expr":  "up",
					},
				},
			},
		},
	}

	result, err := skill.CreateDashboardHandler(context.Background(), args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var dashboard map[string]any
	if err := json.Unmarshal([]byte(result), &dashboard); err != nil {
		t.Fatalf("Expected valid JSON result, got error: %v", err)
	}

	dashboardData, ok := dashboard["dashboard"].(map[string]any)
	if !ok {
		t.Error("Expected dashboard object in result")
	}

	title, ok := dashboardData["title"].(string)
	if !ok || title != "Test Dashboard" {
		t.Errorf("Expected title 'Test Dashboard', got %v", title)
	}
}

func TestCreateDashboardHandler_MissingTitle(t *testing.T) {
	logger := zap.NewNop()
	mockGrafana := &mockGrafanaService{}
	config := &config.GrafanaConfig{}

	skill := &CreateDashboardSkill{
		logger:  logger,
		grafana: mockGrafana,
		config:  config,
	}

	args := map[string]any{
		"panels": []any{
			map[string]any{
				"title": "Test Panel",
			},
		},
	}

	_, err := skill.CreateDashboardHandler(context.Background(), args)
	if err == nil {
		t.Error("Expected error for missing dashboard_title")
	}

	expectedError := "dashboard_title is required and must be a string"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestCreateDashboardHandler_MissingPanels(t *testing.T) {
	logger := zap.NewNop()
	mockGrafana := &mockGrafanaService{}
	config := &config.GrafanaConfig{}

	skill := &CreateDashboardSkill{
		logger:  logger,
		grafana: mockGrafana,
		config:  config,
	}

	args := map[string]any{
		"dashboard_title": "Test Dashboard",
	}

	_, err := skill.CreateDashboardHandler(context.Background(), args)
	if err == nil {
		t.Error("Expected error for missing panels")
	}

	expectedError := "panels are required"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestCreateDashboardHandler_DeploymentDisabled(t *testing.T) {
	logger := zap.NewNop()
	mockGrafana := &mockGrafanaService{}
	config := &config.GrafanaConfig{
		DeployEnabled: false,
	}

	skill := &CreateDashboardSkill{
		logger:  logger,
		grafana: mockGrafana,
		config:  config,
	}

	args := map[string]any{
		"dashboard_title": "Test Dashboard",
		"deploy":          true,
		"grafana_url":     "http://test.grafana",
		"panels": []any{
			map[string]any{
				"title": "Test Panel",
			},
		},
	}

	_, err := skill.CreateDashboardHandler(context.Background(), args)
	if err == nil {
		t.Error("Expected error when deployment is disabled but deploy=true")
	}

	expectedError := "grafana deployment is disabled - set GRAFANA_DEPLOY_ENABLED=true to enable dashboard deployments"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestExtractTags(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		expected []string
	}{
		{
			name: "valid tags",
			args: map[string]any{
				"tags": []any{"monitoring", "production", "alerts"},
			},
			expected: []string{"monitoring", "production", "alerts"},
		},
		{
			name:     "no tags",
			args:     map[string]any{},
			expected: []string{},
		},
		{
			name: "mixed types in tags",
			args: map[string]any{
				"tags": []any{"valid", 123, "another"},
			},
			expected: []string{"valid", "another"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTags(tt.args)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d tags, got %d", len(tt.expected), len(result))
			}

			for i, tag := range tt.expected {
				if i >= len(result) || result[i] != tag {
					t.Errorf("Expected tag[%d] = %s, got %s", i, tag, result[i])
				}
			}
		})
	}
}

func TestExtractTimeRange(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		expected map[string]string
	}{
		{
			name: "valid time range",
			args: map[string]any{
				"time_range": map[string]any{
					"from": "now-1h",
					"to":   "now",
				},
			},
			expected: map[string]string{
				"from": "now-1h",
				"to":   "now",
			},
		},
		{
			name: "partial time range",
			args: map[string]any{
				"time_range": map[string]any{
					"from": "now-2h",
				},
			},
			expected: map[string]string{
				"from": "now-2h",
				"to":   "now",
			},
		},
		{
			name: "no time range",
			args: map[string]any{},
			expected: map[string]string{
				"from": "now-6h",
				"to":   "now",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTimeRange(tt.args)

			if result["from"] != tt.expected["from"] {
				t.Errorf("Expected from = %s, got %s", tt.expected["from"], result["from"])
			}

			if result["to"] != tt.expected["to"] {
				t.Errorf("Expected to = %s, got %s", tt.expected["to"], result["to"])
			}
		})
	}
}

func TestExtractRefreshInterval(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		expected string
	}{
		{
			name: "valid refresh interval",
			args: map[string]any{
				"refresh_interval": "30s",
			},
			expected: "30s",
		},
		{
			name:     "no refresh interval",
			args:     map[string]any{},
			expected: "5s",
		},
		{
			name: "empty refresh interval",
			args: map[string]any{
				"refresh_interval": "",
			},
			expected: "5s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRefreshInterval(tt.args)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetStringOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		m            map[string]any
		key          string
		defaultValue string
		expected     string
	}{
		{
			name: "valid string value",
			m: map[string]any{
				"key": "value",
			},
			key:          "key",
			defaultValue: "default",
			expected:     "value",
		},
		{
			name:         "missing key",
			m:            map[string]any{},
			key:          "key",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name: "empty string value",
			m: map[string]any{
				"key": "",
			},
			key:          "key",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name: "non-string value",
			m: map[string]any{
				"key": 123,
			},
			key:          "key",
			defaultValue: "default",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStringOrDefault(tt.m, tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
