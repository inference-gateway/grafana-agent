# Configuration

All configuration is supplied through environment variables. The tables below
cover the settings this agent reads directly; the project
[README](../README.md#configuration) documents the full A2A server variable
list (`A2A_*`).

## LLM client

The agent defers to an OpenAI-compatible LLM client.

| Variable | Description | Default |
|----------|-------------|---------|
| `A2A_AGENT_CLIENT_PROVIDER` | Provider: `openai`, `anthropic`, `azure`, `ollama`, `deepseek` | |
| `A2A_AGENT_CLIENT_MODEL` | Model identifier | |
| `A2A_AGENT_CLIENT_API_KEY` | Provider API key | |
| `A2A_AGENT_CLIENT_BASE_URL` | Custom endpoint, e.g. an Inference Gateway | |

## Prometheus

The `discover_metrics`, `generate_promql_queries`, and `validate_promql_query`
tools each take a `prometheus_url` argument on the call. When a request does
not specify one, the agent's system prompt falls back to a default of
`http://prometheus.grafana-agent.svc.cluster.local:9090`. The
[Kubernetes example](../examples/kubernetes/README.md) also exposes a matching
`PROMETHEUS_URL` environment variable so deployments can advertise the endpoint
in one place.

## Grafana

The `create_dashboard` and `deploy_dashboard` tools read these settings from
`spec.config.grafana` in `agent.yaml` (env prefix `GRAFANA_`). Deployment is
disabled by default, so a dashboard is only pushed to Grafana when you
explicitly opt in.

| Variable | Description | Default |
|----------|-------------|---------|
| `GRAFANA_URL` | Grafana base URL (Cloud or self-hosted) | |
| `GRAFANA_API_KEY` | Grafana API key / service-account token | |
| `GRAFANA_ORG_ID` | Grafana organisation ID | |
| `GRAFANA_DEPLOY_ENABLED` | Allow `deploy_dashboard` / `create_dashboard` to push to Grafana | `false` |

Deploying a dashboard requires both `GRAFANA_DEPLOY_ENABLED=true` and a
configured `GRAFANA_API_KEY`; the tools return an error otherwise. A
`grafana_url` argument on the tool call overrides `GRAFANA_URL` for that
request.

## Telemetry

OpenTelemetry instrumentation is enabled by default via `spec.telemetry` in
`agent.yaml`. Metrics are exposed on a Prometheus endpoint; traces can be
exported via OTLP when configured.

| Variable | Description | Default |
|----------|-------------|---------|
| `A2A_TELEMETRY_ENABLE` | Enable OpenTelemetry instrumentation | `true` |
| `A2A_OTEL_METRICS_EXPORTER` | Metrics exporter (`prometheus`, `otlp`, or `none`) | `prometheus` |
| `A2A_OTEL_EXPORTER_PROMETHEUS_PORT` | Prometheus metrics endpoint port | `9464` |
| `A2A_OTEL_TRACES_EXPORTER` | Trace exporter (`otlp` or `none`) | `none` |

Set `A2A_TELEMETRY_ENABLE=false` to disable telemetry entirely. The Prometheus
metrics endpoint is served at `0.0.0.0:<port>/metrics` when the Prometheus
exporter is active.

## Built-in tools

| Variable | Description | Default |
|----------|-------------|---------|
| `TOOLS_READ_ENABLED` | Enable the built-in `read` tool used to load skill playbooks | `true` |
