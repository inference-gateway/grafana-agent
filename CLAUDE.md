# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

grafana-agent is an A2A (Agent-to-Agent) server implementing the [A2A Protocol](https://github.com/inference-gateway/adk) for agent-to-agent communication. A2A agent server for grafana dashboards automation tasks. The project is automatically generated from ADL (Agent Definition Language) specifications defined in `agent.yaml`.

## Core Architecture

### ADL-Generated Structure

The codebase is generated using ADL CLI 0.39.1 and follows a strict generation pattern:
- **Generated Files**: Marked with `DO NOT EDIT` headers - manual changes will be overwritten
- **Configuration Source**: `agent.yaml` - defines agent capabilities, skills, and metadata
- **Server Implementation**: Built on the ADK (Agent Development Kit) framework from `github.com/inference-gateway/adk`

### Key Components

- **Main Entry Point**: `main.go` - A cobra-based CLI. The root command exposes
  `--version` and `--help`; the `start` subcommand boots the A2A server with:
  - OpenAI-compatible LLM client configuration
  - Agent builder with system prompt from `agent.yaml`
  - A2A server with streaming and background task handlers
  - Graceful shutdown handling

- **Agent Configuration**: `.well-known/agent-card.json` - Serves agent metadata at runtime
- **Environment Configuration**: Extensive env vars with `A2A_` prefix (see README for full list)

## Development Commands

```bash
# Generate/regenerate code from ADL specification
task generate

# Run the agent in development mode (debug enabled, port 8080)
task run

# Run tests (note: no tests currently exist)
task test
task test:cover  # with coverage

# Code quality
task lint         # Run golangci-lint
task fmt          # Format code with go fmt

# Build
task build        # Creates bin/grafana-agent
task docker:build # Build Docker image

# Clean build artifacts
task clean
```

## Testing Individual Components

```bash
# Run specific test file (when tests are added)
go test -v ./path/to/package -run TestFunctionName

# Debug with A2A Debugger
docker run --rm -it --network host ghcr.io/inference-gateway/a2a-debugger:latest \
  --server-url http://localhost:8080 tasks submit "Your query"
```

## LLM Provider Configuration

The agent uses OpenAI-compatible LLM client. Configure with:
- `A2A_AGENT_CLIENT_PROVIDER`: `openai`, `anthropic`, `azure`, `ollama`, `deepseek`
- `A2A_AGENT_CLIENT_MODEL`: Model identifier
- `A2A_AGENT_CLIENT_API_KEY`: Provider API key
- `A2A_AGENT_CLIENT_BASE_URL`: Custom endpoint (optional)

## Adding New Functionality

### Tools (function-call)
The following tools are currently defined:
- **Read** (built-in): Read a file from disk. Returns its contents, optionally sliced by line offset/limit. Use this to load SKILL.md bodies on demand.
- **discover_metrics**: Discovers available metrics from a Prometheus endpoint with optional filtering
- **generate_promql_queries**: Generates PromQL query suggestions for given metric names by querying Prometheus metadata
- **validate_promql_query**: Validates a PromQL query against a Prometheus server
- **create_dashboard**: Creates a Grafana dashboard with specified panels, queries, and configurations
- **deploy_dashboard**: Deploys a dashboard JSON to Grafana (Cloud or self-hosted)

To modify tools:
1. Update `agent.yaml` `spec.tools` with tool definitions
2. Run `task generate` to regenerate the codebase
3. Implement tool logic in the generated `tools/` files (look for TODO placeholders)
4. Write tests for each tool

### Skills (markdown system-prompt playbooks)
The following skills are currently shipped with the agent:
- **promql** (registry): Write, validate, and optimise PromQL queries for Prometheus and Grafana Cloud Metrics. Use when the user asks to query metrics, write a PromQL expression, calculate rates, aggregate across labels, build histogram quantiles, create recording rules, debug query performance, or understand metric cardinality. Triggers on phrases like "PromQL", "Prometheus query", "write a metric query", "calculate rate", "histogram_quantile", "recording rule", "metric cardinality", "sum by", "rate vs irate", "absent()", or "query is slow".
- **dashboarding** (registry): Create, modify, and organise Grafana dashboards including panels, variables, transformations, and alerting. Use when the user asks to create a Grafana dashboard, add a panel, configure a time series or stat panel, add template variables, set up dashboard linking, use transformations, configure thresholds, build a dashboard for a service, or export dashboard JSON. Triggers on phrases like "create dashboard", "add panel", "time series panel", "Grafana dashboard JSON", "template variables", "dashboard variable", "panel transformation", "threshold", "stat panel", "table panel", "Grafana annotations", or "dashboard folder".

Each skill lives in its own directory at `skills/<id>/SKILL.md` and is
loaded into the system prompt at startup. Bare skills can ship arbitrary
bundled assets (scripts, templates, resources) alongside `SKILL.md` -
the whole `skills/<id>/` directory is protected by `.adl-ignore` against
regeneration overwrites. To modify skills:
1. Update `agent.yaml` `spec.skills` with skill definitions
2. Run `task generate` (registry skills are re-fetched; bare skill
   directories are preserved when listed in `.adl-ignore`)
3. For bare skills, edit `skills/<id>/SKILL.md` directly - frontmatter
   (`name`/`description`/`tags`) shows up on the agent card. Drop helper
   scripts or templates next to it (e.g. `skills/<id>/scripts/foo.py`).

### Modifying Agent Behavior

- **System Prompt**: Edit in `agent.yaml`, then regenerate
- **Capabilities**: Modify in `agent.yaml` (streaming, pushNotifications, stateTransitionHistory)
- **Server Configuration**: Update environment variables or `agent.yaml` server section

## Testing Strategy

When implementing tests:
- Create `*_test.go` files alongside implementation files
- Use table-driven tests for comprehensive coverage
- Mock external dependencies (LLM client, Redis if used)
- Test A2A protocol compliance with integration tests

## Environment Management

### Development Environment
- **Flox Environment**: ✅ Configured via `.flox/env/manifest.toml` providing Go 1.26.2, linter, `go-task`, Docker, and the Claude Code CLI. Activate with `flox activate`.
- **Docker Compose**: ✅ Local service stack defined in `docker-compose.yaml`. Bring up the Inference Gateway and the agent (built from the local `Dockerfile`) with `docker compose up --build`. Opt-in profiles add the `infer` CLI (`docker compose --profile cli run --rm cli`) and the `a2a-debugger` (`docker compose --profile debugger run --rm debugger --server-url http://grafana-agent:8080 tasks list`).

## Important Constraints

- **Generated Files**: Never manually edit files with "DO NOT EDIT" headers
- **Configuration Changes**: Always modify `agent.yaml` and regenerate
- **ADL Version**: Ensure ADL CLI 0.39.1 or compatible version for regeneration
- **Port Configuration**: Default 8080, configurable via `A2A_PORT` or `A2A_SERVER_PORT`

## Debugging Tips

- Enable debug mode: `A2A_DEBUG=true`
- Check health: `GET /health`
- View agent metadata: `GET /.well-known/agent-card.json`
- Monitor streaming updates: Set `A2A_STREAMING_STATUS_UPDATE_INTERVAL`
- Use A2A Debugger container for interactive testing
