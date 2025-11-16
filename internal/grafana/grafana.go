package grafana

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	config "github.com/inference-gateway/grafana-agent/config"
	zap "go.uber.org/zap"
)

// Dashboard represents a Grafana dashboard
type Dashboard struct {
	Dashboard map[string]any `json:"dashboard"`
	FolderUID string         `json:"folderUid"`
	Message   string         `json:"message"`
	Overwrite bool          `json:"overwrite"`
}

// DashboardResponse represents the response from dashboard creation
type DashboardResponse struct {
	ID      int    `json:"id"`
	UID     string `json:"uid"`
	URL     string `json:"url"`
	Status  string `json:"status"`
	Version int    `json:"version"`
	Slug    string `json:"slug"`
}

// Grafana represents the grafana service interface
type Grafana interface {
	CreateDashboard(ctx context.Context, dashboard Dashboard, grafanaURL, apiKey string) (*DashboardResponse, error)
	UpdateDashboard(ctx context.Context, dashboard Dashboard, grafanaURL, apiKey string) (*DashboardResponse, error)
	GetDashboard(ctx context.Context, uid, grafanaURL, apiKey string) (*Dashboard, error)
	DeleteDashboard(ctx context.Context, uid, grafanaURL, apiKey string) error
}

// grafanaImpl is the implementation of Grafana
type grafanaImpl struct {
	logger *zap.Logger
	client *http.Client
}

// NewGrafanaService creates a new instance of Grafana
func NewGrafanaService(logger *zap.Logger, cfg *config.Config) (Grafana, error) {
	logger.Info("initializing grafana service")
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	return &grafanaImpl{
		logger: logger,
		client: client,
	}, nil
}

// CreateDashboard creates a new dashboard in Grafana
func (g *grafanaImpl) CreateDashboard(ctx context.Context, dashboard Dashboard, grafanaURL, apiKey string) (*DashboardResponse, error) {
	url := fmt.Sprintf("%s/api/dashboards/db", strings.TrimRight(grafanaURL, "/"))
	
	jsonData, err := json.Marshal(dashboard)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal dashboard: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create dashboard: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("grafana returned status %d", resp.StatusCode)
	}

	var dashboardResp DashboardResponse
	if err := json.NewDecoder(resp.Body).Decode(&dashboardResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	g.logger.Info("Dashboard created successfully", 
		zap.Int("id", dashboardResp.ID),
		zap.String("uid", dashboardResp.UID),
		zap.String("url", dashboardResp.URL))

	return &dashboardResp, nil
}

// UpdateDashboard updates an existing dashboard in Grafana
func (g *grafanaImpl) UpdateDashboard(ctx context.Context, dashboard Dashboard, grafanaURL, apiKey string) (*DashboardResponse, error) {
	// Set overwrite to true for updates
	dashboard.Overwrite = true
	return g.CreateDashboard(ctx, dashboard, grafanaURL, apiKey)
}

// GetDashboard retrieves a dashboard from Grafana
func (g *grafanaImpl) GetDashboard(ctx context.Context, uid, grafanaURL, apiKey string) (*Dashboard, error) {
	url := fmt.Sprintf("%s/api/dashboards/uid/%s", strings.TrimRight(grafanaURL, "/"), uid)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("dashboard not found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("grafana returned status %d", resp.StatusCode)
	}

	var response struct {
		Dashboard map[string]any `json:"dashboard"`
		Meta      map[string]any `json:"meta"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &Dashboard{
		Dashboard: response.Dashboard,
	}, nil
}

// DeleteDashboard deletes a dashboard from Grafana
func (g *grafanaImpl) DeleteDashboard(ctx context.Context, uid, grafanaURL, apiKey string) error {
	url := fmt.Sprintf("%s/api/dashboards/uid/%s", strings.TrimRight(grafanaURL, "/"), uid)
	
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete dashboard: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("grafana returned status %d", resp.StatusCode)
	}

	g.logger.Info("Dashboard deleted successfully", zap.String("uid", uid))
	return nil
}
