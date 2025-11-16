package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	server "github.com/inference-gateway/adk/server"
	config "github.com/inference-gateway/grafana-agent/config"
	grafana "github.com/inference-gateway/grafana-agent/internal/grafana"
	zap "go.uber.org/zap"
)

// CreateDashboardSkill struct holds the skill with services
type CreateDashboardSkill struct {
	logger  *zap.Logger
	grafana grafana.Grafana
	config  *config.GrafanaConfig
}

// NewCreateDashboardSkill creates a new create_dashboard skill
func NewCreateDashboardSkill(logger *zap.Logger, grafana grafana.Grafana, grafanaConfig *config.GrafanaConfig) server.Tool {
	skill := &CreateDashboardSkill{
		logger:  logger,
		grafana: grafana,
		config:  grafanaConfig,
	}
	return server.NewBasicTool(
		"create_dashboard",
		"Creates a Grafana dashboard with specified panels, queries, and configurations",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"dashboard_title": map[string]any{
					"description": "The title of the Grafana dashboard",
					"type":        "string",
				},
				"description": map[string]any{
					"description": "Description of what the dashboard monitors or displays",
					"type":        "string",
				},
				"grafana_url": map[string]any{
					"description": "Grafana server URL (overrides default configuration if provided)",
					"type":        "string",
				},
				"deploy": map[string]any{
					"description": "Whether to deploy the dashboard to Grafana (requires grafana_url and GRAFANA_DEPLOY_ENABLED=true)",
					"type":        "boolean",
				},
				"panels": map[string]any{
					"description": "Array of panel configurations (title, type, queries, etc.)",
					"items":       map[string]any{"type": "object"},
					"type":        "array",
				},
				"refresh_interval": map[string]any{
					"description": "Auto-refresh interval (e.g., \"5s\", \"1m\", \"5m\")",
					"type":        "string",
				},
				"tags": map[string]any{
					"description": "Tags to categorize the dashboard",
					"items":       map[string]any{"type": "string"},
					"type":        "array",
				},
				"time_range": map[string]any{
					"description": "Default time range for the dashboard (from, to)",
					"properties":  map[string]any{"from": map[string]any{"type": "string"}, "to": map[string]any{"type": "string"}},
					"type":        "object",
				},
				"variables": map[string]any{
					"description": "Dashboard template variables for dynamic queries",
					"items":       map[string]any{"type": "object"},
					"type":        "array",
				},
			},
			"required": []string{"dashboard_title", "panels"},
		},
		skill.CreateDashboardHandler,
	)
}

// CreateDashboardHandler handles the create_dashboard skill execution
func (s *CreateDashboardSkill) CreateDashboardHandler(ctx context.Context, args map[string]any) (string, error) {
	dashboardTitle, ok := args["dashboard_title"].(string)
	if !ok || dashboardTitle == "" {
		return "", fmt.Errorf("dashboard_title is required and must be a string")
	}

	panels, ok := args["panels"].([]any)
	if !ok || len(panels) == 0 {
		return "", fmt.Errorf("panels are required")
	}

	deploy, deployRequested := args["deploy"].(bool)
	if deployRequested && deploy {
		if s.config != nil && !s.config.DeployEnabled {
			log.Printf("WARNING: Grafana deployment attempted but GRAFANA_DEPLOY_ENABLED=false")
			return "", fmt.Errorf("grafana deployment is disabled - set GRAFANA_DEPLOY_ENABLED=true to enable dashboard deployments")
		}

		var grafanaURL string
		if urlParam, ok := args["grafana_url"].(string); ok && urlParam != "" {
			grafanaURL = urlParam
		} else if s.config != nil && s.config.URL != "" {
			grafanaURL = s.config.URL
		}

		if grafanaURL == "" {
			return "", fmt.Errorf("deployment requested but no grafana_url provided")
		}
	}

	var grafanaURL string
	if urlParam, ok := args["grafana_url"].(string); ok && urlParam != "" {
		grafanaURL = urlParam
	} else if s.config != nil && s.config.URL != "" {
		grafanaURL = s.config.URL
	}

	if grafanaURL != "" {
		log.Printf("INFO: Using Grafana URL: %s", grafanaURL)
	}
	if s.config != nil && s.config.APIKey != "" {
		log.Printf("INFO: Grafana API key configured")
	}

	dashboard := map[string]any{
		"dashboard": map[string]any{
			"title":                dashboardTitle,
			"tags":                 extractTags(args),
			"timezone":             "browser",
			"panels":               processPanels(panels),
			"time":                 extractTimeRange(args),
			"refresh":              extractRefreshInterval(args),
			"schemaVersion":        36,
			"version":              0,
			"editable":             true,
			"fiscalYearStartMonth": 0,
			"graphTooltip":         0,
			"links":                []any{},
			"liveNow":              false,
		},
		"folderUid": "",
		"message":   "",
		"overwrite": false,
	}

	if description, ok := args["description"].(string); ok && description != "" {
		dashboard["dashboard"].(map[string]any)["description"] = description
	}

	if variables, ok := args["variables"].([]any); ok && len(variables) > 0 {
		dashboard["dashboard"].(map[string]any)["templating"] = map[string]any{
			"list": processVariables(variables),
		}
	}

	if deployRequested && deploy {
		var grafanaURL string
		var apiKey string

		if urlParam, ok := args["grafana_url"].(string); ok && urlParam != "" {
			grafanaURL = urlParam
		} else if s.config != nil && s.config.URL != "" {
			grafanaURL = s.config.URL
		}

		if s.config != nil && s.config.APIKey != "" {
			apiKey = s.config.APIKey
		}

		if apiKey == "" {
			return "", fmt.Errorf("deployment requested but no API key configured - set GRAFANA_API_KEY")
		}

		grafanaDashboard := grafana.Dashboard{
			Dashboard: dashboard["dashboard"].(map[string]any),
			FolderUID: "",
			Message:   "Dashboard created via grafana-agent",
			Overwrite: true,
		}

		resp, err := s.grafana.CreateDashboard(ctx, grafanaDashboard, grafanaURL, apiKey)
		if err != nil {
			return "", fmt.Errorf("failed to deploy dashboard to Grafana: %w", err)
		}

		s.logger.Info("Dashboard deployed successfully",
			zap.String("grafana_url", grafanaURL),
			zap.String("dashboard_uid", resp.UID),
			zap.Int("dashboard_id", resp.ID))

		deploymentInfo := map[string]any{
			"status":      "deployed",
			"grafana_url": grafanaURL,
			"dashboard": map[string]any{
				"id":  resp.ID,
				"uid": resp.UID,
				"url": resp.URL,
			},
			"dashboard_json": dashboard,
		}

		jsonBytes, err := json.MarshalIndent(deploymentInfo, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal deployment info JSON: %w", err)
		}

		return string(jsonBytes), nil
	}

	jsonBytes, err := json.MarshalIndent(dashboard, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal dashboard JSON: %w", err)
	}

	return string(jsonBytes), nil
}

func extractTags(args map[string]any) []string {
	tags := []string{}
	if tagsRaw, ok := args["tags"].([]any); ok {
		for _, tag := range tagsRaw {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}
	return tags
}

// extractTimeRange extracts time range or returns defaults
func extractTimeRange(args map[string]any) map[string]string {
	defaultTimeRange := map[string]string{
		"from": "now-6h",
		"to":   "now",
	}

	if timeRange, ok := args["time_range"].(map[string]any); ok {
		result := make(map[string]string)
		if from, ok := timeRange["from"].(string); ok && from != "" {
			result["from"] = from
		} else {
			result["from"] = defaultTimeRange["from"]
		}
		if to, ok := timeRange["to"].(string); ok && to != "" {
			result["to"] = to
		} else {
			result["to"] = defaultTimeRange["to"]
		}
		return result
	}

	return defaultTimeRange
}

// extractRefreshInterval extracts refresh interval or returns default
func extractRefreshInterval(args map[string]any) string {
	if refresh, ok := args["refresh_interval"].(string); ok && refresh != "" {
		return refresh
	}
	return "5s"
}

// processPanels converts panel definitions to Grafana panel format
func processPanels(panels []any) []any {
	result := []any{}

	for i, panelRaw := range panels {
		panelMap, ok := panelRaw.(map[string]any)
		if !ok {
			continue
		}

		panel := map[string]any{
			"id":          i + 1,
			"type":        getStringOrDefault(panelMap, "type", "timeseries"),
			"title":       getStringOrDefault(panelMap, "title", fmt.Sprintf("Panel %d", i+1)),
			"gridPos":     extractGridPos(panelMap, i),
			"targets":     extractTargets(panelMap),
			"options":     extractOptions(panelMap),
			"fieldConfig": extractFieldConfig(panelMap),
		}

		if description, ok := panelMap["description"].(string); ok && description != "" {
			panel["description"] = description
		}

		result = append(result, panel)
	}

	return result
}

// extractGridPos extracts grid position or calculates default
func extractGridPos(panel map[string]any, index int) map[string]any {
	if gridPos, ok := panel["gridPos"].(map[string]any); ok {
		return gridPos
	}

	row := index / 2
	col := (index % 2) * 12

	return map[string]any{
		"x": col,
		"y": row * 8,
		"w": 12,
		"h": 8,
	}
}

// extractTargets extracts query targets from panel
func extractTargets(panel map[string]any) []any {
	if targets, ok := panel["targets"].([]any); ok {
		return targets
	}

	return []any{
		map[string]any{
			"refId": "A",
			"expr":  "",
		},
	}
}

// extractOptions extracts panel options
func extractOptions(panel map[string]any) map[string]any {
	if options, ok := panel["options"].(map[string]any); ok {
		return options
	}

	return map[string]any{
		"legend": map[string]any{
			"displayMode": "list",
			"placement":   "bottom",
		},
	}
}

// extractFieldConfig extracts field configuration
func extractFieldConfig(panel map[string]any) map[string]any {
	if fieldConfig, ok := panel["fieldConfig"].(map[string]any); ok {
		return fieldConfig
	}

	return map[string]any{
		"defaults": map[string]any{
			"color": map[string]any{
				"mode": "palette-classic",
			},
			"custom": map[string]any{
				"drawStyle":         "line",
				"lineInterpolation": "linear",
				"fillOpacity":       0,
			},
		},
		"overrides": []any{},
	}
}

// processVariables converts variable definitions to Grafana template variables
func processVariables(variables []any) []any {
	result := []any{}

	for _, varRaw := range variables {
		varMap, ok := varRaw.(map[string]any)
		if !ok {
			continue
		}

		variable := map[string]any{
			"name":  getStringOrDefault(varMap, "name", "var"),
			"type":  getStringOrDefault(varMap, "type", "query"),
			"label": getStringOrDefault(varMap, "label", ""),
		}

		if query, ok := varMap["query"].(string); ok && query != "" {
			variable["query"] = query
		}

		if datasource, ok := varMap["datasource"].(string); ok && datasource != "" {
			variable["datasource"] = datasource
		}

		result = append(result, variable)
	}

	return result
}

// getStringOrDefault safely extracts a string value or returns default
func getStringOrDefault(m map[string]any, key, defaultValue string) string {
	if val, ok := m[key].(string); ok && val != "" {
		return val
	}
	return defaultValue
}
