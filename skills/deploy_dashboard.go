package skills

import (
	"context"
	"encoding/json"
	"fmt"

	server "github.com/inference-gateway/adk/server"
	config "github.com/inference-gateway/grafana-agent/config"
	grafana "github.com/inference-gateway/grafana-agent/internal/grafana"
	zap "go.uber.org/zap"
)

// DeployDashboardSkill struct holds the skill with services
type DeployDashboardSkill struct {
	logger        *zap.Logger
	grafanaSvc    grafana.Grafana
	grafanaConfig *config.GrafanaConfig
}

// NewDeployDashboardSkill creates a new deploy_dashboard skill
func NewDeployDashboardSkill(logger *zap.Logger, grafanaSvc grafana.Grafana, grafanaConfig *config.GrafanaConfig) server.Tool {
	skill := &DeployDashboardSkill{
		logger:        logger,
		grafanaSvc:    grafanaSvc,
		grafanaConfig: grafanaConfig,
	}
	return server.NewBasicTool(
		"deploy_dashboard",
		"Deploys a dashboard JSON to Grafana (Cloud or self-hosted)",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"dashboard_json": map[string]any{
					"description": "The complete dashboard JSON object to deploy",
					"type":        "object",
				},
				"folder_uid": map[string]any{
					"description": "Optional folder UID where the dashboard should be deployed",
					"type":        "string",
				},
				"grafana_url": map[string]any{
					"description": "Grafana server URL (user provides in prompt or uses config default)",
					"type":        "string",
				},
				"message": map[string]any{
					"description": "Optional commit message describing the dashboard changes",
					"type":        "string",
				},
				"overwrite": map[string]any{
					"description": "Whether to overwrite an existing dashboard with the same UID (default true)",
					"type":        "boolean",
				},
			},
			"required": []string{"dashboard_json"},
		},
		skill.DeployDashboardHandler,
	)
}

// DeployDashboardHandler handles the deploy_dashboard skill execution
func (s *DeployDashboardSkill) DeployDashboardHandler(ctx context.Context, args map[string]any) (string, error) {
	if s.grafanaConfig != nil && !s.grafanaConfig.DeployEnabled {
		s.logger.Warn("Grafana deployment attempted but GRAFANA_DEPLOY_ENABLED=false")
		return "", fmt.Errorf("grafana deployment is disabled - set GRAFANA_DEPLOY_ENABLED=true to enable dashboard deployments")
	}

	dashboardJSON, ok := args["dashboard_json"].(map[string]any)
	if !ok || len(dashboardJSON) == 0 {
		return "", fmt.Errorf("dashboard_json is required and must be a valid object")
	}

	var grafanaURL string
	if urlParam, ok := args["grafana_url"].(string); ok && urlParam != "" {
		grafanaURL = urlParam
	} else if s.grafanaConfig != nil && s.grafanaConfig.URL != "" {
		grafanaURL = s.grafanaConfig.URL
	}

	if grafanaURL == "" {
		return "", fmt.Errorf("grafana_url must be provided either as a parameter or in configuration (GRAFANA_URL)")
	}

	var apiKey string
	if s.grafanaConfig != nil && s.grafanaConfig.APIKey != "" {
		apiKey = s.grafanaConfig.APIKey
	}

	if apiKey == "" {
		return "", fmt.Errorf("grafana API key is required - set GRAFANA_API_KEY")
	}

	folderUID := ""
	if uid, ok := args["folder_uid"].(string); ok {
		folderUID = uid
	}

	overwrite := true
	if ow, ok := args["overwrite"].(bool); ok {
		overwrite = ow
	}

	message := "Dashboard deployed via grafana-agent"
	if msg, ok := args["message"].(string); ok && msg != "" {
		message = msg
	}

	dashboard := grafana.Dashboard{
		Dashboard: dashboardJSON,
		FolderUID: folderUID,
		Message:   message,
		Overwrite: overwrite,
	}

	s.logger.Info("Deploying dashboard to Grafana",
		zap.String("grafana_url", grafanaURL),
		zap.String("folder_uid", folderUID),
		zap.Bool("overwrite", overwrite))

	resp, err := s.grafanaSvc.CreateDashboard(ctx, dashboard, grafanaURL, apiKey)
	if err != nil {
		return "", fmt.Errorf("failed to deploy dashboard to Grafana: %w", err)
	}

	s.logger.Info("Dashboard deployed successfully",
		zap.String("grafana_url", grafanaURL),
		zap.String("dashboard_uid", resp.UID),
		zap.Int("dashboard_id", resp.ID),
		zap.String("dashboard_url", resp.URL))

	result := map[string]any{
		"status":      "deployed",
		"grafana_url": grafanaURL,
		"dashboard": map[string]any{
			"id":      resp.ID,
			"uid":     resp.UID,
			"url":     resp.URL,
			"version": resp.Version,
			"slug":    resp.Slug,
		},
		"message": message,
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal deployment result: %w", err)
	}

	return string(jsonBytes), nil
}
