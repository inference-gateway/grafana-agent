package promql

import (
	"context"

	config "github.com/inference-gateway/grafana-agent/config"
	zap "go.uber.org/zap"
)

//go:generate go tool counterfeiter -generate

// PromQL represents the promql service interface
// PromQL service for building and validating Prometheus queries
//
//counterfeiter:generate . PromQL
type PromQL interface {
	// GetMetricMetadata fetches metadata for a specific metric from Prometheus
	GetMetricMetadata(ctx context.Context, prometheusURL, metricName string) (*MetricInfo, error)

	// GenerateQueries generates appropriate PromQL queries based on metric type and name
	GenerateQueries(metricInfo *MetricInfo) []QuerySuggestion

	// ValidateQuery validates a PromQL query against Prometheus
	ValidateQuery(ctx context.Context, prometheusURL, query string) error

	// GetBestQuery selects the most appropriate query for visualization
	GetBestQuery(suggestions []QuerySuggestion) QuerySuggestion
}

// promqlImpl is the implementation of PromQL
type promqlImpl struct {
	logger *zap.Logger
}

// NewPromQLService creates a new instance of PromQL
func NewPromQLService(logger *zap.Logger, cfg *config.Config) (PromQL, error) {
	logger.Info("initializing promql service")

	return &promqlImpl{
		logger: logger,
	}, nil
}

// GetMetricMetadata fetches metadata for a specific metric from Prometheus
func (p *promqlImpl) GetMetricMetadata(ctx context.Context, prometheusURL, metricName string) (*MetricInfo, error) {
	p.logger.Debug("fetching metric metadata",
		zap.String("metric", metricName),
		zap.String("prometheus_url", prometheusURL))

	client := newPrometheusClient(prometheusURL)
	return client.getMetricMetadata(ctx, metricName)
}

// GenerateQueries generates appropriate PromQL queries based on metric type and name
func (p *promqlImpl) GenerateQueries(metricInfo *MetricInfo) []QuerySuggestion {
	p.logger.Debug("generating queries",
		zap.String("metric", metricInfo.Name),
		zap.String("type", string(metricInfo.Type)))

	return generateQueries(metricInfo)
}

// ValidateQuery validates a PromQL query against Prometheus
func (p *promqlImpl) ValidateQuery(ctx context.Context, prometheusURL, query string) error {
	p.logger.Debug("validating query",
		zap.String("query", query),
		zap.String("prometheus_url", prometheusURL))

	client := newPrometheusClient(prometheusURL)
	return client.validateQuery(ctx, query)
}

// GetBestQuery selects the most appropriate query for visualization
func (p *promqlImpl) GetBestQuery(suggestions []QuerySuggestion) QuerySuggestion {
	p.logger.Debug("selecting best query",
		zap.Int("suggestion_count", len(suggestions)))

	return getBestQuery(suggestions)
}
