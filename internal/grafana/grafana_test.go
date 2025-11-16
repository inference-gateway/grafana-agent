package grafana

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/inference-gateway/grafana-agent/config"
	"go.uber.org/zap"
)

func TestNewGrafanaService(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{}

	service, err := NewGrafanaService(logger, cfg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if service == nil {
		t.Error("Expected non-nil service")
	}
}

func TestCreateDashboard(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name           string
		dashboard      Dashboard
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		expectedUID    string
		expectedID     int
		validateFunc   func(t *testing.T, resp *DashboardResponse)
	}{
		{
			name: "successful dashboard creation",
			dashboard: Dashboard{
				Dashboard: map[string]any{
					"title": "Test Dashboard",
				},
				Message:   "Created via API",
				Overwrite: false,
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.Header.Get("Authorization") != "Bearer test-api-key" {
					t.Errorf("Expected Authorization header with Bearer token")
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type: application/json")
				}

				var received Dashboard
				if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
					t.Fatalf("Failed to decode request body: %v", err)
				}

				w.WriteHeader(http.StatusOK)
				response := DashboardResponse{
					ID:      123,
					UID:     "test-uid-123",
					URL:     "/d/test-uid-123/test-dashboard",
					Status:  "success",
					Version: 1,
					Slug:    "test-dashboard",
				}
				json.NewEncoder(w).Encode(response)
			},
			wantErr:     false,
			expectedUID: "test-uid-123",
			expectedID:  123,
			validateFunc: func(t *testing.T, resp *DashboardResponse) {
				if resp.URL != "/d/test-uid-123/test-dashboard" {
					t.Errorf("Expected URL '/d/test-uid-123/test-dashboard', got %s", resp.URL)
				}
				if resp.Status != "success" {
					t.Errorf("Expected status 'success', got %s", resp.Status)
				}
			},
		},
		{
			name: "grafana returns error status",
			dashboard: Dashboard{
				Dashboard: map[string]any{
					"title": "Test Dashboard",
				},
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Invalid dashboard",
				})
			},
			wantErr: true,
		},
		{
			name: "grafana returns invalid JSON",
			dashboard: Dashboard{
				Dashboard: map[string]any{
					"title": "Test Dashboard",
				},
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("invalid json"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			service, _ := NewGrafanaService(logger, &config.Config{})

			resp, err := service.CreateDashboard(context.Background(), tt.dashboard, server.URL, "test-api-key")

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if resp.UID != tt.expectedUID {
				t.Errorf("Expected UID %s, got %s", tt.expectedUID, resp.UID)
			}

			if resp.ID != tt.expectedID {
				t.Errorf("Expected ID %d, got %d", tt.expectedID, resp.ID)
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, resp)
			}
		})
	}
}

func TestUpdateDashboard(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name            string
		dashboard       Dashboard
		expectOverwrite bool
		serverResponse  func(w http.ResponseWriter, r *http.Request)
		wantErr         bool
	}{
		{
			name: "update sets overwrite to true",
			dashboard: Dashboard{
				Dashboard: map[string]any{
					"title": "Updated Dashboard",
					"uid":   "existing-uid",
				},
				Overwrite: false,
			},
			expectOverwrite: true,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				var received Dashboard
				if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
					t.Fatalf("Failed to decode request body: %v", err)
				}

				if !received.Overwrite {
					t.Error("Expected Overwrite to be true for update")
				}

				w.WriteHeader(http.StatusOK)
				response := DashboardResponse{
					ID:      123,
					UID:     "existing-uid",
					URL:     "/d/existing-uid/updated-dashboard",
					Status:  "success",
					Version: 2,
				}
				json.NewEncoder(w).Encode(response)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			service, _ := NewGrafanaService(logger, &config.Config{})

			resp, err := service.UpdateDashboard(context.Background(), tt.dashboard, server.URL, "test-api-key")

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if resp == nil {
				t.Fatal("Expected non-nil response")
			}
		})
	}
}

func TestGetDashboard(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name           string
		uid            string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		expectedError  string
		validateFunc   func(t *testing.T, dashboard *Dashboard)
	}{
		{
			name: "successful dashboard retrieval",
			uid:  "test-uid",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				if r.Header.Get("Authorization") != "Bearer test-api-key" {
					t.Errorf("Expected Authorization header with Bearer token")
				}

				w.WriteHeader(http.StatusOK)
				response := map[string]any{
					"dashboard": map[string]any{
						"title": "Existing Dashboard",
						"uid":   "test-uid",
					},
					"meta": map[string]any{
						"version": 1,
					},
				}
				json.NewEncoder(w).Encode(response)
			},
			wantErr: false,
			validateFunc: func(t *testing.T, dashboard *Dashboard) {
				if dashboard.Dashboard["title"] != "Existing Dashboard" {
					t.Errorf("Expected title 'Existing Dashboard', got %v", dashboard.Dashboard["title"])
				}
			},
		},
		{
			name: "dashboard not found",
			uid:  "nonexistent-uid",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Dashboard not found",
				})
			},
			wantErr:       true,
			expectedError: "dashboard not found",
		},
		{
			name: "grafana returns server error",
			uid:  "test-uid",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
		{
			name: "invalid JSON response",
			uid:  "test-uid",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("invalid json"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			service, _ := NewGrafanaService(logger, &config.Config{})

			dashboard, err := service.GetDashboard(context.Background(), tt.uid, server.URL, "test-api-key")

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if tt.expectedError != "" && err.Error() != tt.expectedError {
					t.Errorf("Expected error '%s', got '%s'", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, dashboard)
			}
		})
	}
}

func TestDeleteDashboard(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name           string
		uid            string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
	}{
		{
			name: "successful dashboard deletion",
			uid:  "test-uid",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" {
					t.Errorf("Expected DELETE request, got %s", r.Method)
				}
				if r.Header.Get("Authorization") != "Bearer test-api-key" {
					t.Errorf("Expected Authorization header with Bearer token")
				}

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Dashboard deleted",
				})
			},
			wantErr: false,
		},
		{
			name: "grafana returns error status",
			uid:  "test-uid",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: true,
		},
		{
			name: "grafana returns server error",
			uid:  "test-uid",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			service, _ := NewGrafanaService(logger, &config.Config{})

			err := service.DeleteDashboard(context.Background(), tt.uid, server.URL, "test-api-key")

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}
		})
	}
}
