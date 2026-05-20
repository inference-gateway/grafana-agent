---
name: metric-exploration
description: Use this when the user is exploring what metrics a Prometheus server exposes or wants candidate PromQL expressions before committing to a dashboard. Combines discover_metrics, generate_promql_queries, and validate_promql_query into a discovery loop.
tags:
  - prometheus
  - promql
  - exploration
---

# metric-exploration

Use this when the user wants to find their way around a Prometheus
endpoint - listing metrics, getting query ideas, or sanity-checking
expressions - without committing to a dashboard yet.

## When to use

Trigger this skill on requests like:

- "What metrics does this Prometheus expose for the API gateway?"
- "Show me everything that starts with `http_`"
- "Give me a few query ideas for `process_cpu_seconds_total`"
- "Does `rate(foo_total[5m])` work against this server?"
- "What does this histogram look like? Suggest a couple of useful queries."

If the user is ready to assemble a dashboard from these metrics, switch
to [[dashboard-authoring]].

## Workflow

1. **Confirm the Prometheus endpoint.** Default to the `PROMETHEUS_URL`
   environment variable when the user doesn't specify one.

2. **Enumerate metrics.** Call `discover_metrics`. Use `name_pattern`
   (a regex) to scope wide queries and `metric_type` to filter by
   `counter`, `gauge`, `histogram`, or `summary` when the user has
   indicated the shape they care about. Surface a concise summary
   (name, type, help text) rather than dumping the full list.

3. **Draft expressions on demand.** For metrics the user wants to
   explore further, call `generate_promql_queries` once with all of
   them listed in `metric_names`. Present the suggestions grouped by
   metric, and highlight the visualization type the generator
   recommends.

4. **Validate before recommending.** Whenever you suggest a query the
   user might actually run, call `validate_promql_query` against the
   same endpoint first. Treat a validation failure as a signal to
   refine the expression (often a missing label, a typo in the metric
   name, or a bracket window that doesn't apply to instant vectors).
   Only present validated queries as "ready to use."

5. **Stay exploratory.** Don't call `create_dashboard` or
   `deploy_dashboard` from this skill. If the user pivots to building
   a dashboard, hand off to [[dashboard-authoring]].

## Tools

In typical invocation order:

1. `discover_metrics` - list metrics on the Prometheus endpoint
2. `generate_promql_queries` - draft candidate expressions for interesting metrics
3. `validate_promql_query` - confirm a candidate expression parses before recommending it
