---
name: dashboard-authoring
description: Use this when the user asks to build, modify, or deploy a Grafana dashboard. Walks the full lifecycle - discover_metrics to enumerate signals, generate_promql_queries to draft expressions, validate_promql_query to confirm they parse, create_dashboard to assemble the JSON, and deploy_dashboard to ship it.
tags:
  - grafana
  - dashboard
  - workflow
---

# dashboard-authoring

Use this when the user asks to build, modify, preview, or deploy a Grafana
dashboard backed by Prometheus metrics.

## When to use

Trigger this skill on requests like:

- "Build a dashboard for the checkout service"
- "I need panels showing p95 latency and error rate for API X"
- "Generate a dashboard JSON I can paste into Grafana"
- "Deploy this dashboard to our staging Grafana"
- "What does a SLO dashboard for service Y look like?"

If the user just wants to look around at what metrics exist without ending
up with a dashboard, use [[metric-exploration]] instead.

## Workflow

1. **Clarify the goal.** Confirm the service or subsystem the dashboard
   should cover, the Prometheus endpoint to query, and whether the user
   wants the JSON back or a deployment. Default `prometheus_url` to the
   value of `PROMETHEUS_URL`; default `grafana_url` to `GRAFANA_URL` if
   the user does not provide one.

2. **Discover candidate metrics.** Call `discover_metrics` against the
   Prometheus endpoint. Use the optional `name_pattern` regex to scope
   the result (e.g. `^http_.*` for HTTP signals, `^.*_seconds$` for
   latencies) and `metric_type` to filter for `counter`, `gauge`,
   `histogram`, or `summary` when the dashboard intent makes the
   metric shape obvious.

3. **Propose a panel plan.** From the discovered metrics, pick a small
   set that covers the user's intent (rate, error rate, latency
   percentiles, saturation). Surface this plan to the user before
   spending tokens generating queries for metrics they don't want.

4. **Draft PromQL.** For each chosen metric name, call
   `generate_promql_queries` once with the full list in `metric_names`.
   The tool returns several suggestions per metric along with a
   recommended visualization type - pick the suggestion that matches
   the panel's purpose (timeseries for rates, stat for headlines,
   gauge for current values, heatmap for histogram buckets).

5. **Validate each query.** Call `validate_promql_query` on every
   expression you intend to ship. Do not skip this step even for
   suggestions that came from `generate_promql_queries` - the
   generator builds expressions from metric metadata, but only the
   Prometheus server can confirm they parse against the live schema.
   Drop or fix any expression that fails validation before assembling
   the dashboard.

6. **Assemble the dashboard.** Call `create_dashboard` with
   `dashboard_title`, `panels` (one panel object per validated
   expression, with `title`, `type`, and a `targets` array containing
   `refId` and `expr`), plus optional `description`, `tags`,
   `time_range`, `refresh_interval`, and `variables`. The tool returns
   the full Grafana dashboard JSON in the response body. Default
   `time_range` is `now-6h` to `now` and default `refresh_interval` is
   `5s` if you don't specify them.

7. **Deploy on request.** If the user explicitly asks to deploy and
   `GRAFANA_DEPLOY_ENABLED=true`, call `deploy_dashboard` with the
   `dashboard` object from step 6's result. `create_dashboard` also
   supports an inline `deploy: true` flag, but prefer the two-step
   flow so the user can review the JSON before it ships. Honor the
   user's `folder_uid`, `overwrite`, and `message` if they supply
   them; otherwise the tool defaults overwrite to `true` and writes
   a sensible commit message.

## Tools

In typical invocation order:

1. `discover_metrics` - enumerate Prometheus metrics matching the dashboard's scope
2. `generate_promql_queries` - draft candidate expressions for the chosen metrics
3. `validate_promql_query` - confirm each expression parses against Prometheus
4. `create_dashboard` - assemble panels into a Grafana dashboard JSON
5. `deploy_dashboard` - ship the JSON to Grafana (only when the user asks)

## Output expectations

`create_dashboard` returns the dashboard JSON as a string in the tool
result, not as a separate artifact. When the user wants to install the
dashboard manually, paste the JSON back to them; when they want to ship
it, feed the JSON's `dashboard` object straight into `deploy_dashboard`.
