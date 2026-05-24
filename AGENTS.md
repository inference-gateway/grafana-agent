# AGENTS.md

This file describes the agents available in this A2A (Agent-to-Agent) system.

## Agent Overview

### grafana-agent
**Version**: 0.2.1  
**Description**: A2A agent server for grafana dashboards automation tasks

This agent is built using the Agent Definition Language (ADL) and provides A2A communication capabilities.

## Agent Capabilities
- **Streaming**: ✅ Real-time response streaming supported
- **Push Notifications**: ❌ Server-sent events not supported
- **State History**: ❌ State transition history not tracked

## AI Configuration

**System Prompt**: You are a Grafana expert. Your role is to guide users in designing highly effective, visually clear, and actionable dashboards.
You provide best practices for data visualization, panel configuration, query optimization, alerting, and overall dashboard usability.
Always offer practical examples and explain the reasoning behind your recommendations.

When using Prometheus-related tools:
- Use the PROMETHEUS_URL environment variable for prometheus_url parameters (default: http://prometheus.grafana-agent.svc.cluster.local:9090)
- The Prometheus server is available at this internal Kubernetes service URL

When using Grafana-related tools:
- Use the GRAFANA_URL environment variable for grafana_url parameters if not explicitly provided by the user


**Configuration:**

## Tools

This agent exposes 6 function-call tools:

### Read (built-in)
- **Description**: Read a file from disk. Returns its contents, optionally sliced by line offset/limit. Use this to load SKILL.md bodies on demand.
- **Parameters**: file_path, offset, limit

### discover_metrics
- **Description**: Discovers available metrics from a Prometheus endpoint with optional filtering
- **Tags**: promql, prometheus, metrics, discovery
- **Input Schema**: Defined in agent configuration
- **Output Schema**: Defined in agent configuration

### generate_promql_queries
- **Description**: Generates PromQL query suggestions for given metric names by querying Prometheus metadata
- **Tags**: promql, prometheus, query, metrics
- **Input Schema**: Defined in agent configuration
- **Output Schema**: Defined in agent configuration

### validate_promql_query
- **Description**: Validates a PromQL query against a Prometheus server
- **Tags**: promql, prometheus, validation
- **Input Schema**: Defined in agent configuration
- **Output Schema**: Defined in agent configuration

### create_dashboard
- **Description**: Creates a Grafana dashboard with specified panels, queries, and configurations
- **Tags**: grafana, dashboard, visualization
- **Input Schema**: Defined in agent configuration
- **Output Schema**: Defined in agent configuration

### deploy_dashboard
- **Description**: Deploys a dashboard JSON to Grafana (Cloud or self-hosted)
- **Tags**: grafana, dashboard, deployment
- **Input Schema**: Defined in agent configuration
- **Output Schema**: Defined in agent configuration

## Skills

This agent ships 2 markdown skills that are loaded into the system prompt at startup:

### promql
- **Description**: Write, validate, and optimise PromQL queries for Prometheus and Grafana Cloud Metrics. Use when the user asks to query metrics, write a PromQL expression, calculate rates, aggregate across labels, build histogram quantiles, create recording rules, debug query performance, or understand metric cardinality. Triggers on phrases like "PromQL", "Prometheus query", "write a metric query", "calculate rate", "histogram_quantile", "recording rule", "metric cardinality", "sum by", "rate vs irate", "absent()", or "query is slow".
- **Version**: 6311c4f4d36db3c5a85686ef2b3ce5fed4e53c0c
- **Source**: fetched from the skills registry (`skills/promql/SKILL.md`)

### dashboarding
- **Description**: Create, modify, and organise Grafana dashboards including panels, variables, transformations, and alerting. Use when the user asks to create a Grafana dashboard, add a panel, configure a time series or stat panel, add template variables, set up dashboard linking, use transformations, configure thresholds, build a dashboard for a service, or export dashboard JSON. Triggers on phrases like "create dashboard", "add panel", "time series panel", "Grafana dashboard JSON", "template variables", "dashboard variable", "panel transformation", "threshold", "stat panel", "table panel", "Grafana annotations", or "dashboard folder".
- **Version**: 6311c4f4d36db3c5a85686ef2b3ce5fed4e53c0c
- **Source**: fetched from the skills registry (`skills/dashboarding/SKILL.md`)

## Server Configuration

**Port**: 8080
**Debug Mode**: ❌ Disabled
**Authentication**: ❌ Not required

## API Endpoints

The agent exposes the following HTTP endpoints:

- `GET /.well-known/agent-card.json` - Agent metadata and capabilities
- `GET /health` - Health check endpoint
- `POST /a2a` - JSON-RPC endpoint for all A2A operations (skill execution, streaming, etc.)

## Environment Setup

### Required Environment Variables

Key environment variables you'll need to configure:
- `PORT` - Server port (configured: 8080)

### Development Environment
- **Flox Environment**: ✅ Configured for reproducible development setup (`flox activate`)
- **Docker Compose**: ✅ Local service stack defined in `docker-compose.yaml` - brings up the Inference Gateway and this agent (built from `Dockerfile`) with `docker compose up --build`. Opt-in profiles `cli` and `debugger` expose the `infer` CLI and `a2a-debugger` for end-to-end testing.

## Usage

### Starting the Agent

```bash
# Install dependencies
go mod download

# Run the agent (the binary is a CLI; `start` boots the server)
go run . start

# Or use Task
task run
```

### Communicating with the Agent

The agent implements the A2A protocol and can be communicated with via HTTP requests:

```bash
# Get agent information
curl http://localhost:8080/.well-known/agent-card.json
```

Refer to the main README.md for specific skill execution examples and input schemas.

## Deployment

**Deployment Type**: Manual
- Build and run the agent binary directly
- Use provided Dockerfile for containerized deployment

### Docker Deployment

```bash
# Build image
docker build -t grafana-agent .

# Run container
docker run -p 8080:8080 grafana-agent
```

## Development

### Project Structure

```
.
├── main.go                       # Server entry point
├── tools/                        # Function-call tools
│   └── read.go                   # Read a file from disk. Returns its contents, optionally sliced by line offset/limit. Use this to load SKILL.md bodies on demand.
│   └── discover_metrics.go       # Discovers available metrics from a Prometheus endpoint with optional filtering
│   └── generate_promql_queries.go# Generates PromQL query suggestions for given metric names by querying Prometheus metadata
│   └── validate_promql_query.go  # Validates a PromQL query against a Prometheus server
│   └── create_dashboard.go       # Creates a Grafana dashboard with specified panels, queries, and configurations
│   └── deploy_dashboard.go       # Deploys a dashboard JSON to Grafana (Cloud or self-hosted)
├── skills/                       # Skill directories (SKILL.md + optional assets)
│   └── promql/                   # Write, validate, and optimise PromQL queries for Prometheus and Grafana Cloud Metrics. Use when the user asks to query metrics, write a PromQL expression, calculate rates, aggregate across labels, build histogram quantiles, create recording rules, debug query performance, or understand metric cardinality. Triggers on phrases like "PromQL", "Prometheus query", "write a metric query", "calculate rate", "histogram_quantile", "recording rule", "metric cardinality", "sum by", "rate vs irate", "absent()", or "query is slow".
│       └── SKILL.md              # Playbook prepended to the system prompt
│   └── dashboarding/             # Create, modify, and organise Grafana dashboards including panels, variables, transformations, and alerting. Use when the user asks to create a Grafana dashboard, add a panel, configure a time series or stat panel, add template variables, set up dashboard linking, use transformations, configure thresholds, build a dashboard for a service, or export dashboard JSON. Triggers on phrases like "create dashboard", "add panel", "time series panel", "Grafana dashboard JSON", "template variables", "dashboard variable", "panel transformation", "threshold", "stat panel", "table panel", "Grafana annotations", or "dashboard folder".
│       └── SKILL.md              # Playbook prepended to the system prompt
├── .well-known/                  # Agent configuration
│   └── agent-card.json           # Agent metadata
├── go.mod                        # Go module definition
└── README.md                     # Project documentation
```

### Testing

```bash
# Run tests
task test
go test ./...

# Run with coverage
task test:coverage
```

## Contributing

1. Implement business logic in skill files (replace TODO placeholders)
2. Add comprehensive tests for new functionality
3. Follow the established code patterns and conventions
4. Ensure proper error handling throughout
5. Update documentation as needed

## Agent Metadata

This agent was generated using ADL CLI v0.2.1 with the following configuration:

- **Language**: Go
- **Template**: Minimal A2A Agent
- **ADL Version**: adl.inference-gateway.com/v1

---

For more information about A2A agents and the ADL specification, visit the [ADL CLI documentation](https://github.com/inference-gateway/adl-cli).
