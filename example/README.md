# Grafana Agent Example

This example demonstrates how to use the grafana-agent for AI-powered Grafana dashboard automation. The agent can create dashboards, configure panels, set up queries, and more using natural language.

## Architecture

The example includes a complete monitoring stack:

- **Grafana Agent**: A2A server that automates Grafana dashboard operations
- **Grafana**: Visualization and monitoring platform (http://localhost:3000)
- **Prometheus**: Time-series database for metrics (http://localhost:9090)
- **Demo OTEL Service**: Sample service generating metrics with OpenTelemetry instrumentation (http://localhost:8082)
- **Inference Gateway**: Routes LLM requests to configured providers
- **CLI**: Interactive command-line interface for agent interaction
- **A2A Debugger**: Tool for debugging and testing agent tasks

## Prerequisites

Configure the environment variables:

```bash
cp .env.example .env
```

**Note:** Add at least one LLM provider API key (e.g., DeepSeek, Google, Anthropic, or OpenAI) in the `.env` file.

## Quick Start

1. **Start all services:**
   ```bash
   docker compose up --build
   ```

2. **Access the services:**
   - Grafana: http://localhost:3000 (admin/admin)
   - Prometheus: http://localhost:9090
   - Demo Service: http://localhost:8082
   - Demo Service Metrics: http://localhost:8082/metrics

3. **Verify the demo dashboard:**
   Navigate to Grafana and check for the pre-provisioned "Demo OTEL Service Dashboard"

## Usage

### Interactive Chat Mode

Use the CLI to interact with the agent:

```bash
docker compose run --rm cli
```

### Example Prompts

#### Create a Simple Dashboard

```text
Please create a dashboard called "System Overview" with panels showing CPU usage and memory usage from the demo-service. Use the agent.
```

The agent will:
- Connect to Grafana using configured credentials
- Create a new dashboard with the specified name
- Add panels with queries for CPU and memory metrics
- Configure appropriate visualizations

#### Create an Advanced Dashboard

```text
Create a dashboard named "HTTP Performance" with the following panels:
1. Request rate by endpoint (time series)
2. P95 and P99 latency (time series)
3. Error rate (graph)
4. Active connections gauge
Use data from the demo-service in Prometheus. Use the agent.
```

The agent will:
- Create a multi-panel dashboard
- Configure PromQL queries for each metric
- Set up appropriate visualization types (time series, gauge)
- Apply labels and formatting

#### Modify an Existing Dashboard

```text
Add a new panel to the "Demo OTEL Service Dashboard" showing the processing queue size as a stat panel. Use the agent.
```

#### Create a Dashboard from Template

```text
Create a comprehensive monitoring dashboard for the demo-service with all available metrics organized by category: HTTP metrics, System metrics, and Queue metrics. Use the agent.
```

### Direct A2A API Usage

You can also interact with the agent directly via the A2A debugger:

```bash
# List all tasks
docker compose run --rm a2a-debugger tasks list

# Submit a task
docker compose run --rm a2a-debugger tasks submit "Create a dashboard showing HTTP request rates"

# Get task status
docker compose run --rm a2a-debugger tasks get <task-id>
```

## Demo Service Metrics

The demo OTEL service exports the following Prometheus metrics:

### Traditional Prometheus Metrics
- `http_requests_total` - Counter: Total HTTP requests by method, endpoint, and status
- `http_request_duration_seconds` - Histogram: Request duration distribution
- `active_connections` - Gauge: Current number of active connections
- `processing_queue_size` - Gauge: Current processing queue size
- `errors_total` - Counter: Total number of errors

### OpenTelemetry Metrics
- `cpu_usage_percent` - Gauge: Simulated CPU usage percentage
- `memory_usage_bytes` - Gauge: Simulated memory usage in bytes
- `request_latency_ms` - Histogram: Request latency in milliseconds

The service simulates realistic traffic patterns with various HTTP methods, endpoints, and response codes every 5 seconds.

## Configuration

### Grafana Credentials

Default credentials are set in `.env`:
- Username: `admin`
- Password: `admin`

You can also use a Grafana API key by setting `GRAFANA_API_KEY` in the `.env` file.

### LLM Provider Selection

Configure which LLM provider to use in `.env`:

```bash
# Use DeepSeek (recommended for cost-effectiveness)
A2A_AGENT_CLIENT_PROVIDER=deepseek
A2A_AGENT_CLIENT_MODEL=deepseek-chat

# Use Google Gemini
A2A_AGENT_CLIENT_PROVIDER=google
A2A_AGENT_CLIENT_MODEL=gemini-2.0-flash-exp

# Use Anthropic Claude
A2A_AGENT_CLIENT_PROVIDER=anthropic
A2A_AGENT_CLIENT_MODEL=claude-3-5-sonnet-20241022

# Use OpenAI GPT
A2A_AGENT_CLIENT_PROVIDER=openai
A2A_AGENT_CLIENT_MODEL=gpt-4o
```

### Prometheus Configuration

The Prometheus configuration (`prometheus.yml`) includes scrape configs for:
- Prometheus itself (self-monitoring)
- Demo OTEL service (application metrics)
- Grafana (platform metrics)

Metrics are scraped every 15 seconds by default.

### Grafana Provisioning

Grafana is pre-configured with:
- **Datasource**: Prometheus datasource at `http://prometheus:9090`
- **Dashboard**: Demo OTEL Service Dashboard with pre-configured panels

Configuration files:
- `provisioning/datasources/prometheus.yml` - Datasource configuration
- `provisioning/dashboards/dashboard.yml` - Dashboard provider configuration
- `provisioning/dashboards/demo-service-dashboard.json` - Sample dashboard definition

## Development

### Modify the Agent

To make changes to the agent:

1. Edit `agent.yaml` to modify skills or behavior
2. Run `task generate` to regenerate code
3. Rebuild and restart:
   ```bash
   docker compose up --build agent
   ```

### Add Custom Metrics

To add custom metrics to the demo service:

1. Edit `demo-service/main.go`
2. Add new Prometheus or OTEL metric definitions
3. Update the simulation logic in `simulateMetrics()`
4. Rebuild:
   ```bash
   docker compose up --build demo-service
   ```

### Create Custom Dashboards

You can manually create dashboards in Grafana and export them as JSON:

1. Create dashboard in Grafana UI
2. Go to Dashboard Settings → JSON Model
3. Copy the JSON
4. Save to `provisioning/dashboards/your-dashboard.json`
5. Restart Grafana to load it

## Troubleshooting

### Agent Cannot Connect to Grafana

Check the agent logs:
```bash
docker compose logs agent
```

Verify Grafana credentials in `.env` match Grafana configuration.

### No Metrics in Prometheus

1. Check Prometheus targets: http://localhost:9090/targets
2. Verify all targets are "UP"
3. Check demo-service logs:
   ```bash
   docker compose logs demo-service
   ```

### Dashboards Not Loading

1. Check Grafana provisioning logs:
   ```bash
   docker compose logs grafana
   ```

2. Verify provisioning files are mounted correctly:
   ```bash
   docker compose exec grafana ls -la /etc/grafana/provisioning/dashboards/
   ```

### LLM Provider Errors

1. Verify API key is set correctly in `.env`
2. Check inference-gateway logs:
   ```bash
   docker compose logs inference-gateway
   ```
3. Ensure the provider is supported by inference-gateway

## Example Workflow

Here's a complete workflow demonstrating the agent's capabilities:

1. **Start the stack:**
   ```bash
   docker compose up --build
   ```

2. **Verify metrics are being collected:**
   - Open http://localhost:9090
   - Go to "Graph" and query: `http_requests_total`
   - Verify data is returned

3. **Check the pre-provisioned dashboard:**
   - Open http://localhost:3000
   - Navigate to "Dashboards"
   - Open "Demo OTEL Service Dashboard"
   - Verify panels are showing data

4. **Use the agent to create a new dashboard:**
   ```bash
   docker compose run --rm cli
   ```

   Then enter:
   ```text
   Create a new dashboard called "API Monitoring" with these panels:
   1. HTTP request rate by status code (stacked area chart)
   2. Average request duration (line chart)
   3. Error rate percentage (stat panel with red threshold above 5%)
   Use metrics from demo-service. Use the agent.
   ```

5. **Verify the new dashboard:**
   - Return to Grafana
   - Find the "API Monitoring" dashboard
   - Verify all panels are configured correctly

6. **Clean up:**
   ```bash
   docker compose down
   ```

## Advanced Features

### Custom PromQL Queries

The agent can generate complex PromQL queries. Try:

```text
Create a dashboard with a panel showing the 95th percentile request latency grouped by endpoint, calculated over 5-minute windows. Use the agent.
```

### Dashboard Templating

Request dashboards with variables:

```text
Create a dashboard with a variable selector for the service name, and show CPU and memory metrics filtered by the selected service. Use the agent.
```

### Annotations and Alerts

Ask the agent to configure annotations or alert rules:

```text
Add an annotation to the dashboard that marks when error rates exceed 10 requests per minute. Use the agent.
```

## Clean Up

Stop and remove all containers:

```bash
docker compose down
```

Remove volumes (this will delete all Grafana dashboards and Prometheus data):

```bash
docker compose down -v
```

## Additional Resources

- [Grafana Documentation](https://grafana.com/docs/)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [A2A Protocol](https://github.com/inference-gateway/adk)
- [Inference Gateway](https://github.com/inference-gateway/inference-gateway)
