# Getting Started

`grafana-agent` is an [A2A](https://github.com/inference-gateway/adk) server
that turns natural-language requests into Prometheus queries and Grafana
dashboards. This guide gets a local instance running and answering requests.

## Prerequisites

- Go 1.26.4+ (or Docker) to run the agent.
- An API key for an LLM provider (`openai`, `anthropic`, `azure`, `ollama`,
  or `deepseek`).
- A reachable Prometheus server — required by the metric and PromQL tools.
- A Grafana instance with an API key — only needed to deploy dashboards.

## Run the agent

The generated binary is a CLI; `start` boots the A2A server.

```bash
# From source
go run . start

# Or build the CLI and run it
task build
./bin/grafana-agent start

# Or with Docker
docker build -t grafana-agent .
docker run -p 8080:8080 grafana-agent
```

The server listens on port `8080` by default (override with `A2A_PORT`).

## Connect an LLM

The agent needs an LLM provider to reason over requests:

```bash
export A2A_AGENT_CLIENT_PROVIDER=deepseek
export A2A_AGENT_CLIENT_MODEL=deepseek-v4-flash
export A2A_AGENT_CLIENT_API_KEY=<your-key>
```

See [Configuration](configuration.md) for the full list of variables,
including the Prometheus and Grafana settings the tools rely on.

## Send your first request

Confirm the agent is healthy and inspect its capabilities:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/.well-known/agent-card.json
```

Then submit a task with the
[A2A Debugger](https://github.com/inference-gateway/a2a-debugger):

```bash
docker run --rm -it --network host \
  ghcr.io/inference-gateway/a2a-debugger:latest \
  --server-url http://localhost:8080 \
  tasks submit "Discover the HTTP metrics available in Prometheus"
```

For a full local stack (agent + Grafana + Prometheus + a demo service), see the
[docker-compose example](../examples/docker-compose/README.md). A Kubernetes
walkthrough lives in the [kubernetes example](../examples/kubernetes/README.md).

## Next steps

- [Configuration](configuration.md) — every environment variable this agent reads.
- [Usage](usage.md) — the discover → query → build → deploy workflow.
