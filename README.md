<div align="center">

# Grafana Agent

[![CI](https://github.com/inference-gateway/grafana-agent/workflows/CI/badge.svg)](https://github.com/inference-gateway/grafana-agent/actions/workflows/ci.yml)
[![Go Report Card](https://img.shields.io/badge/Go%20Report%20Card-A+-brightgreen?style=flat&logo=go&logoColor=white)](https://goreportcard.com/report/github.com/inference-gateway/grafana-agent)
[![Go Version](https://img.shields.io/badge/Go-1.26.4+-00ADD8?style=flat&logo=go)](https://golang.org)
[![A2A Protocol](https://img.shields.io/badge/A2A-Protocol-blue?style=flat)](https://github.com/inference-gateway/adk)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)

**A2A agent server for grafana dashboards automation tasks**

A enterprise-ready [Agent-to-Agent (A2A)](https://github.com/inference-gateway/adk) server that provides AI-powered capabilities through a standardized protocol.

</div>

## Quick Start

The generated binary is a CLI. `start` boots the A2A server; `--help` and
`--version` work as you'd expect.

```bash
# Run the agent
go run . start

# Or build and invoke the CLI directly
task build
./bin/grafana-agent start

# Or with Docker
docker build -t grafana-agent .
docker run -p 8080:8080 grafana-agent
```

### CLI

| Command | Description |
|---------|-------------|
| `grafana-agent start` | Start the A2A server (blocks until SIGINT/SIGTERM) |
| `grafana-agent --help` | Show top-level help (and per-subcommand with `<cmd> --help`) |
| `grafana-agent --version` | Print the embedded version and exit |

## Quick Install

Add this agent to your Inference Gateway CLI:

```bash
infer agents add grafana-agent http://localhost:8080 \
  --oci ghcr.io/inference-gateway/grafana-agent:latest \
  --run
```

## Features

- ✅ A2A protocol compliant
- ✅ AI-powered capabilities
- ✅ Streaming support
- ✅ OpenTelemetry instrumentation
- ✅ Enterprise-ready
- ✅ Minimal dependencies

## Endpoints

- `GET /.well-known/agent-card.json` - Agent metadata and capabilities
- `GET /health` - Health check endpoint
- `POST /a2a` - A2A protocol endpoint

## Available Tools

| Tool | Description | Parameters |
|------|-------------|------------|
| `Read` | Read a file from disk. Returns its contents, optionally sliced by line offset/limit. Use this to load SKILL.md bodies on demand. | file_path, offset, limit |
| `discover_metrics` | Discovers available metrics from a Prometheus endpoint with optional filtering | metric_type, name_pattern, prometheus_url |
| `generate_promql_queries` | Generates PromQL query suggestions for given metric names by querying Prometheus metadata | metric_names, prometheus_url |
| `validate_promql_query` | Validates a PromQL query against a Prometheus server | prometheus_url, query |
| `create_dashboard` | Creates a Grafana dashboard with specified panels, queries, and configurations | dashboard_title, deploy, description, grafana_url, panels, refresh_interval, tags, time_range, variables |
| `deploy_dashboard` | Deploys a dashboard JSON to Grafana (Cloud or self-hosted) | dashboard_json, folder_uid, grafana_url, message, overwrite |

## Examples

| Example | Description |
|---------|-------------|
| [Discover metrics for a service](examples/discover-metrics-for-a-service/) | Ask "What HTTP metrics are exposed in Prometheus matching http_.*?" and the agent uses discover_metrics to list the matching series, optionally filtered by metric type (counter, gauge, histogram, summary). |
| [Build and validate a PromQL query](examples/build-and-validate-a-promql-query/) | Ask "Give me the p99 request latency per endpoint" and the agent drafts PromQL with generate_promql_queries, applies the promql skill's best practices, and confirms it parses against Prometheus with validate_promql_query before returning it. |
| [Create a dashboard for a service](examples/create-a-dashboard-for-a-service/) | Ask "Create a RED-method dashboard for my checkout service" and the agent uses the dashboarding skill and create_dashboard to assemble time series and stat panels wired to validated PromQL queries, with thresholds and template variables. |
| [Deploy a dashboard to Grafana](examples/deploy-a-dashboard-to-grafana/) | Provide a Grafana URL and API key, then ask "Deploy this dashboard to my Grafana Cloud instance" and the agent pushes the dashboard JSON with deploy_dashboard (guarded by GRAFANA_DEPLOY_ENABLED) to Grafana Cloud or a self-hosted instance. |

## Skills (loaded into the system prompt)

| Skill | Description | Source |
|-------|-------------|--------|
| `promql` | Write, validate, and optimise PromQL queries for Prometheus and Grafana Cloud Metrics. Use when the user asks to query metrics, write a PromQL expression, calculate rates, aggregate across labels, build histogram quantiles, create recording rules, debug query performance, or understand metric cardinality. Triggers on phrases like "PromQL", "Prometheus query", "write a metric query", "calculate rate", "histogram_quantile", "recording rule", "metric cardinality", "sum by", "rate vs irate", "absent()", or "query is slow". | registry @ 6311c4f4d36db3c5a85686ef2b3ce5fed4e53c0c |
| `dashboarding` | Create, modify, and organise Grafana dashboards including panels, variables, transformations, and alerting. Use when the user asks to create a Grafana dashboard, add a panel, configure a time series or stat panel, add template variables, set up dashboard linking, use transformations, configure thresholds, build a dashboard for a service, or export dashboard JSON. Triggers on phrases like "create dashboard", "add panel", "time series panel", "Grafana dashboard JSON", "template variables", "dashboard variable", "panel transformation", "threshold", "stat panel", "table panel", "Grafana annotations", or "dashboard folder". | registry @ 6311c4f4d36db3c5a85686ef2b3ce5fed4e53c0c |

## Documentation
- [Getting Started](docs/getting-started.md)
- [Configuration](docs/configuration.md)
- [Usage](docs/usage.md)

## Configuration

The agent is configured via environment variables. Defaults are derived
from `agent.yaml`; see [CONFIGURATIONS.md](CONFIGURATIONS.md) for the
full reference of custom and `A2A_*` variables.

## Development

```bash
# Generate code from ADL
task generate

# Run tests
task test

# Build the application
task build

# Run linter
task lint

# Format code
task fmt
```

### Adding Dependencies

The generator owns the baseline toolchain pins (SDK, server framework,
logging, CLI, sandbox utilities). To extend the project without forking
the templates, declare extras in `agent.yaml` - every empty list below
is rendered by `adl init --defaults` precisely so it's discoverable:

| Where | Purpose | Example entry | Rendered into |
|-------|---------|---------------|---------------|
| `spec.language.go.vendor.deps` | Runtime Go modules | `github.com/stretchr/testify@v1.10.0` | `go.mod` `require` block |
| `spec.language.go.vendor.devdeps` | Executable dev tools (Go 1.24 `tool` directive) | `golang.org/x/tools/cmd/stringer@v0.20.0` | `go.mod` `tool` directive |
| `spec.development.deps` | Cross-cutting sandbox tools (not tied to one language) | `kubectl@1.31.0`, `terraform@1.9.5`, `deno@2.1.4` | Flox `manifest.toml` / devcontainer feature |

Entries use the `<package>@<version>` form. Built-in pins always win on
conflict; the generator prints a warning and skips the user entry when
shadowing is attempted. After editing `agent.yaml`, re-run `task generate`
to refresh the manifests.

### Debugging

Use the [A2A Debugger](https://github.com/inference-gateway/a2a-debugger) to test and debug your A2A agent during development. It provides a web interface for sending requests to your agent and inspecting responses, making it easier to troubleshoot issues and validate your implementation.

```bash
docker run --rm -it --network host ghcr.io/inference-gateway/a2a-debugger:latest --server-url http://localhost:8080 tasks submit "What are your skills?"
```

```bash
docker run --rm -it --network host ghcr.io/inference-gateway/a2a-debugger:latest --server-url http://localhost:8080 tasks list
```

```bash
docker run --rm -it --network host ghcr.io/inference-gateway/a2a-debugger:latest --server-url http://localhost:8080 tasks get <task ID>
```

## Deployment

### Docker

The Docker image can be built with custom version information using build arguments:

```bash
docker build \
  --build-arg VERSION=1.2.3 \
  --build-arg AGENT_NAME="My Custom Agent" \
  --build-arg AGENT_DESCRIPTION="Custom agent description" \
  -t grafana-agent:1.2.3 .
```

**Available Build Arguments:**

- `VERSION` - Agent version (default: `0.3.1`)
- `AGENT_NAME` - Agent name (default: `grafana-agent`)
- `AGENT_DESCRIPTION` - Agent description (default: `A2A agent server for grafana dashboards automation tasks`)

These values are embedded into the binary at build time using linker flags, making them accessible at runtime without requiring environment variables.

## License

Apache 2.0 License - see LICENSE file for details
