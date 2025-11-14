package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

var (
	// Traditional Prometheus metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	activeConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_connections",
			Help: "Number of active connections",
		},
	)

	processingQueueSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "processing_queue_size",
			Help: "Current size of the processing queue",
		},
	)

	errorRate = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "errors_total",
			Help: "Total number of errors",
		},
	)

	// OTEL metrics
	meter metric.Meter
)

func initOTEL() (*sdkmetric.MeterProvider, error) {
	ctx := context.Background()

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(getEnv("OTEL_SERVICE_NAME", "demo-service")),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("environment", "demo"),
		),
	)
	if err != nil {
		return nil, err
	}

	// Create Prometheus exporter
	exporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}

	// Create meter provider
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(exporter),
	)

	// Set global meter provider
	otel.SetMeterProvider(meterProvider)

	// Create meter
	meter = meterProvider.Meter("demo-service")

	return meterProvider, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func simulateMetrics(ctx context.Context) {
	// Create OTEL metrics
	cpuUsage, _ := meter.Float64ObservableGauge(
		"cpu_usage_percent",
		metric.WithDescription("CPU usage percentage"),
	)

	memoryUsage, _ := meter.Float64ObservableGauge(
		"memory_usage_bytes",
		metric.WithDescription("Memory usage in bytes"),
		metric.WithUnit("By"),
	)

	requestLatency, _ := meter.Float64Histogram(
		"request_latency_ms",
		metric.WithDescription("Request latency in milliseconds"),
		metric.WithUnit("ms"),
	)

	// Register callbacks for observable metrics
	_, _ = meter.RegisterCallback(
		func(ctx context.Context, o metric.Observer) error {
			// Simulate CPU usage between 10-80%
			cpu := 10 + rand.Float64()*70
			o.ObserveFloat64(cpuUsage, cpu)

			// Simulate memory usage between 100MB-800MB
			memory := 100_000_000 + rand.Float64()*700_000_000
			o.ObserveFloat64(memoryUsage, memory)

			return nil
		},
		cpuUsage,
		memoryUsage,
	)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Simulate HTTP requests
			methods := []string{"GET", "POST", "PUT", "DELETE"}
			endpoints := []string{"/api/users", "/api/products", "/api/orders", "/api/health"}
			statuses := []string{"200", "201", "400", "404", "500"}

			for i := 0; i < rand.Intn(10)+1; i++ {
				method := methods[rand.Intn(len(methods))]
				endpoint := endpoints[rand.Intn(len(endpoints))]
				status := statuses[rand.Intn(len(statuses))]

				// Record traditional Prometheus metrics
				httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()

				duration := rand.Float64() * 2
				httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration)

				// Record OTEL metrics
				requestLatency.Record(ctx, duration*1000,
					metric.WithAttributes(
						attribute.String("method", method),
						attribute.String("endpoint", endpoint),
						attribute.String("status", status),
					),
				)

				// Occasionally record errors
				if status == "500" || status == "400" {
					errorRate.Inc()
				}
			}

			// Update gauges
			activeConnections.Set(float64(rand.Intn(100) + 10))
			processingQueueSize.Set(float64(rand.Intn(50)))
		}
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	// Initialize OTEL
	meterProvider, err := initOTEL()
	if err != nil {
		log.Fatalf("Failed to initialize OTEL: %v", err)
	}
	defer func() {
		if err := meterProvider.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down meter provider: %v", err)
		}
	}()

	// Start metrics simulation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go simulateMetrics(ctx)

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head><title>Demo OTEL Service</title></head>
<body>
	<h1>Demo OpenTelemetry Service</h1>
	<p>This service exports Prometheus metrics with OTEL instrumentation.</p>
	<ul>
		<li><a href="/metrics">Prometheus Metrics</a></li>
		<li><a href="/health">Health Check</a></li>
	</ul>
	<h2>Available Metrics:</h2>
	<ul>
		<li><strong>http_requests_total</strong> - Total HTTP requests by method, endpoint, and status</li>
		<li><strong>http_request_duration_seconds</strong> - Request duration histogram</li>
		<li><strong>active_connections</strong> - Current number of active connections</li>
		<li><strong>processing_queue_size</strong> - Current processing queue size</li>
		<li><strong>errors_total</strong> - Total number of errors</li>
		<li><strong>cpu_usage_percent</strong> - Simulated CPU usage (OTEL)</li>
		<li><strong>memory_usage_bytes</strong> - Simulated memory usage (OTEL)</li>
		<li><strong>request_latency_ms</strong> - Request latency histogram (OTEL)</li>
	</ul>
</body>
</html>
		`))
	})

	port := getEnv("OTEL_EXPORTER_PROMETHEUS_PORT", "8080")
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Starting demo OTEL service on port %s", port)
		log.Printf("Metrics available at http://localhost:%s/metrics", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-sigChan
	log.Println("Shutting down gracefully...")

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
