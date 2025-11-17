package skills

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/inference-gateway/grafana-agent/config"
	"github.com/inference-gateway/grafana-agent/internal/grafana"
	"go.uber.org/zap"
)

func TestNewDeployDashboardSkill(t *testing.T) {
	logger := zap.NewNop()
	mockGrafana := &mockGrafanaService{}
	config := &config.GrafanaConfig{
		DeployEnabled: true,
		URL:           "http://grafana.test",
		APIKey:        "test-key",
	}

	skill := NewDeployDashboardSkill(logger, mockGrafana, config)

	if skill == nil {
		t.Error("Expected non-nil skill")
	}
}

func TestDeployDashboardHandler_DeploymentDisabled(t *testing.T) {
	logger := zap.NewNop()
	mockGrafana := &mockGrafanaService{}
	config := &config.GrafanaConfig{
		DeployEnabled: false,
	}

	skill := &DeployDashboardSkill{
		logger:        logger,
		grafanaSvc:    mockGrafana,
		grafanaConfig: config,
	}

	args := map[string]any{
		"dashboard_json": map[string]any{
			"title": "Test Dashboard",
		},
		"grafana_url": "http://test.grafana",
	}

	_, err := skill.DeployDashboardHandler(context.Background(), args)
	if err == nil {
		t.Error("Expected error when deployment is disabled")
	}

	expectedError := "grafana deployment is disabled - set GRAFANA_DEPLOY_ENABLED=true to enable dashboard deployments"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestDeployDashboardHandler_MissingDashboardJSON(t *testing.T) {
	logger := zap.NewNop()
	mockGrafana := &mockGrafanaService{}
	config := &config.GrafanaConfig{
		DeployEnabled: true,
	}

	skill := &DeployDashboardSkill{
		logger:        logger,
		grafanaSvc:    mockGrafana,
		grafanaConfig: config,
	}

	args := map[string]any{}

	_, err := skill.DeployDashboardHandler(context.Background(), args)
	if err == nil {
		t.Error("Expected error for missing dashboard_json")
	}

	expectedError := "dashboard_json is required and must be a valid object"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestDeployDashboardHandler_MissingGrafanaURL(t *testing.T) {
	logger := zap.NewNop()
	mockGrafana := &mockGrafanaService{}
	config := &config.GrafanaConfig{
		DeployEnabled: true,
		URL:           "",
	}

	skill := &DeployDashboardSkill{
		logger:        logger,
		grafanaSvc:    mockGrafana,
		grafanaConfig: config,
	}

	args := map[string]any{
		"dashboard_json": map[string]any{
			"title": "Test Dashboard",
		},
	}

	_, err := skill.DeployDashboardHandler(context.Background(), args)
	if err == nil {
		t.Error("Expected error for missing grafana_url")
	}

	expectedError := "grafana_url must be provided either as a parameter or in configuration (GRAFANA_URL)"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestDeployDashboardHandler_MissingAPIKey(t *testing.T) {
	logger := zap.NewNop()
	mockGrafana := &mockGrafanaService{}
	config := &config.GrafanaConfig{
		DeployEnabled: true,
		URL:           "http://grafana.test",
		APIKey:        "",
	}

	skill := &DeployDashboardSkill{
		logger:        logger,
		grafanaSvc:    mockGrafana,
		grafanaConfig: config,
	}

	args := map[string]any{
		"dashboard_json": map[string]any{
			"title": "Test Dashboard",
		},
	}

	_, err := skill.DeployDashboardHandler(context.Background(), args)
	if err == nil {
		t.Error("Expected error for missing API key")
	}

	expectedError := "grafana API key is required - set GRAFANA_API_KEY"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestDeployDashboardHandler_SuccessfulDeployment(t *testing.T) {
	logger := zap.NewNop()
	mockGrafana := &mockGrafanaService{
		createDashboardFunc: func(ctx context.Context, dashboard grafana.Dashboard, grafanaURL, apiKey string) (*grafana.DashboardResponse, error) {
			return &grafana.DashboardResponse{
				ID:      123,
				UID:     "test-uid-123",
				URL:     "/d/test-uid-123/test-dashboard",
				Status:  "success",
				Version: 1,
				Slug:    "test-dashboard",
			}, nil
		},
	}
	config := &config.GrafanaConfig{
		DeployEnabled: true,
		URL:           "http://grafana.test",
		APIKey:        "test-api-key",
	}

	skill := &DeployDashboardSkill{
		logger:        logger,
		grafanaSvc:    mockGrafana,
		grafanaConfig: config,
	}

	args := map[string]any{
		"dashboard_json": map[string]any{
			"title": "Test Dashboard",
			"uid":   "test-uid-123",
		},
	}

	result, err := skill.DeployDashboardHandler(context.Background(), args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Expected valid JSON result, got error: %v", err)
	}

	if status, ok := response["status"].(string); !ok || status != "deployed" {
		t.Errorf("Expected status 'deployed', got %v", status)
	}

	if grafanaURL, ok := response["grafana_url"].(string); !ok || grafanaURL != "http://grafana.test" {
		t.Errorf("Expected grafana_url 'http://grafana.test', got %v", grafanaURL)
	}

	dashboard, ok := response["dashboard"].(map[string]any)
	if !ok {
		t.Fatal("Expected dashboard object in response")
	}

	if id, ok := dashboard["id"].(float64); !ok || int(id) != 123 {
		t.Errorf("Expected dashboard id 123, got %v", id)
	}

	if uid, ok := dashboard["uid"].(string); !ok || uid != "test-uid-123" {
		t.Errorf("Expected dashboard uid 'test-uid-123', got %v", uid)
	}
}

func TestDeployDashboardHandler_WithUserProvidedURL(t *testing.T) {
	logger := zap.NewNop()
	mockGrafana := &mockGrafanaService{
		createDashboardFunc: func(ctx context.Context, dashboard grafana.Dashboard, grafanaURL, apiKey string) (*grafana.DashboardResponse, error) {
			if grafanaURL != "http://user-provided.grafana" {
				t.Errorf("Expected grafanaURL 'http://user-provided.grafana', got %s", grafanaURL)
			}
			return &grafana.DashboardResponse{
				ID:  456,
				UID: "test-uid-456",
				URL: "/d/test-uid-456/test-dashboard",
			}, nil
		},
	}
	config := &config.GrafanaConfig{
		DeployEnabled: true,
		URL:           "http://default.grafana",
		APIKey:        "test-api-key",
	}

	skill := &DeployDashboardSkill{
		logger:        logger,
		grafanaSvc:    mockGrafana,
		grafanaConfig: config,
	}

	args := map[string]any{
		"dashboard_json": map[string]any{
			"title": "Test Dashboard",
		},
		"grafana_url": "http://user-provided.grafana",
	}

	_, err := skill.DeployDashboardHandler(context.Background(), args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestDeployDashboardHandler_WithFolderUID(t *testing.T) {
	logger := zap.NewNop()
	mockGrafana := &mockGrafanaService{
		createDashboardFunc: func(ctx context.Context, dashboard grafana.Dashboard, grafanaURL, apiKey string) (*grafana.DashboardResponse, error) {
			if dashboard.FolderUID != "test-folder-uid" {
				t.Errorf("Expected folderUID 'test-folder-uid', got %s", dashboard.FolderUID)
			}
			return &grafana.DashboardResponse{
				ID:  789,
				UID: "test-uid-789",
				URL: "/d/test-uid-789/test-dashboard",
			}, nil
		},
	}
	config := &config.GrafanaConfig{
		DeployEnabled: true,
		URL:           "http://grafana.test",
		APIKey:        "test-api-key",
	}

	skill := &DeployDashboardSkill{
		logger:        logger,
		grafanaSvc:    mockGrafana,
		grafanaConfig: config,
	}

	args := map[string]any{
		"dashboard_json": map[string]any{
			"title": "Test Dashboard",
		},
		"folder_uid": "test-folder-uid",
	}

	_, err := skill.DeployDashboardHandler(context.Background(), args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestDeployDashboardHandler_WithCustomMessage(t *testing.T) {
	logger := zap.NewNop()
	mockGrafana := &mockGrafanaService{
		createDashboardFunc: func(ctx context.Context, dashboard grafana.Dashboard, grafanaURL, apiKey string) (*grafana.DashboardResponse, error) {
			if dashboard.Message != "Custom deployment message" {
				t.Errorf("Expected message 'Custom deployment message', got %s", dashboard.Message)
			}
			return &grafana.DashboardResponse{
				ID:  999,
				UID: "test-uid-999",
				URL: "/d/test-uid-999/test-dashboard",
			}, nil
		},
	}
	config := &config.GrafanaConfig{
		DeployEnabled: true,
		URL:           "http://grafana.test",
		APIKey:        "test-api-key",
	}

	skill := &DeployDashboardSkill{
		logger:        logger,
		grafanaSvc:    mockGrafana,
		grafanaConfig: config,
	}

	args := map[string]any{
		"dashboard_json": map[string]any{
			"title": "Test Dashboard",
		},
		"message": "Custom deployment message",
	}

	_, err := skill.DeployDashboardHandler(context.Background(), args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestDeployDashboardHandler_WithOverwriteFalse(t *testing.T) {
	logger := zap.NewNop()
	mockGrafana := &mockGrafanaService{
		createDashboardFunc: func(ctx context.Context, dashboard grafana.Dashboard, grafanaURL, apiKey string) (*grafana.DashboardResponse, error) {
			if dashboard.Overwrite != false {
				t.Errorf("Expected overwrite false, got %v", dashboard.Overwrite)
			}
			return &grafana.DashboardResponse{
				ID:  111,
				UID: "test-uid-111",
				URL: "/d/test-uid-111/test-dashboard",
			}, nil
		},
	}
	config := &config.GrafanaConfig{
		DeployEnabled: true,
		URL:           "http://grafana.test",
		APIKey:        "test-api-key",
	}

	skill := &DeployDashboardSkill{
		logger:        logger,
		grafanaSvc:    mockGrafana,
		grafanaConfig: config,
	}

	args := map[string]any{
		"dashboard_json": map[string]any{
			"title": "Test Dashboard",
		},
		"overwrite": false,
	}

	_, err := skill.DeployDashboardHandler(context.Background(), args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestDeployDashboardHandler_DeploymentError(t *testing.T) {
	logger := zap.NewNop()
	mockGrafana := &mockGrafanaService{
		createDashboardFunc: func(ctx context.Context, dashboard grafana.Dashboard, grafanaURL, apiKey string) (*grafana.DashboardResponse, error) {
			return nil, errors.New("grafana API error")
		},
	}
	config := &config.GrafanaConfig{
		DeployEnabled: true,
		URL:           "http://grafana.test",
		APIKey:        "test-api-key",
	}

	skill := &DeployDashboardSkill{
		logger:        logger,
		grafanaSvc:    mockGrafana,
		grafanaConfig: config,
	}

	args := map[string]any{
		"dashboard_json": map[string]any{
			"title": "Test Dashboard",
		},
	}

	_, err := skill.DeployDashboardHandler(context.Background(), args)
	if err == nil {
		t.Error("Expected error from Grafana API")
	}

	expectedError := "failed to deploy dashboard to Grafana: grafana API error"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}
