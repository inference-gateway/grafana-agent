# Usage

`grafana-agent` turns natural-language requests into Prometheus queries and
Grafana dashboards. A typical request flows through four steps, backed by the
agent's tools and skills.

## The workflow

1. **Discover** — `discover_metrics` lists the metrics a Prometheus server
   exposes, optionally filtered by a name regex or metric type (counter, gauge,
   histogram, summary).
2. **Query** — `generate_promql_queries` suggests PromQL for chosen metrics
   using Prometheus metadata, and `validate_promql_query` checks that an
   expression parses against the server. The **promql** skill guides rate
   selection, aggregation, and `histogram_quantile` usage.
3. **Build** — `create_dashboard` assembles a Grafana dashboard from panels,
   queries, thresholds, and template variables. The **dashboarding** skill
   supplies panel and layout best practices.
4. **Deploy** — `deploy_dashboard` (or `create_dashboard` with `deploy: true`)
   pushes the dashboard JSON to Grafana Cloud or a self-hosted instance, gated
   on `GRAFANA_DEPLOY_ENABLED=true` (see [Configuration](configuration.md)).

## Tools

| Tool | Purpose |
|------|---------|
| `discover_metrics` | Discover metrics from a Prometheus endpoint with optional name/type filtering |
| `generate_promql_queries` | Generate PromQL suggestions for given metric names |
| `validate_promql_query` | Validate a PromQL query against Prometheus |
| `create_dashboard` | Build a Grafana dashboard with panels, queries, and variables |
| `deploy_dashboard` | Deploy a dashboard JSON to Grafana (Cloud or self-hosted) |
| `Read` | Load a skill playbook (`SKILL.md`) on demand |

## Skills

Two markdown playbooks are loaded into the system prompt and read on demand:

- **promql** — writing, validating, and optimising PromQL queries.
- **dashboarding** — creating and organising Grafana dashboards: panels,
  variables, transformations, and thresholds.

## Example requests

```text
Discover the HTTP metrics available in Prometheus matching http_.*
Write a PromQL query for the p99 request latency per endpoint
Create a RED-method dashboard for the checkout service
Deploy that dashboard to my Grafana Cloud instance
```

Submit any of these with the A2A Debugger:

```bash
docker run --rm -it --network host \
  ghcr.io/inference-gateway/a2a-debugger:latest \
  --server-url http://localhost:8080 \
  tasks submit "Create a RED-method dashboard for the checkout service"
```

For a runnable end-to-end demo — including a Prometheus instance with live
metrics and a pre-provisioned Grafana — see the
[docker-compose example](../examples/docker-compose/README.md).
