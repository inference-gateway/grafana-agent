# Build and validate a PromQL query

Ask "Give me the p99 request latency per endpoint" and the agent drafts PromQL with generate_promql_queries, applies the promql skill's best practices, and confirms it parses against Prometheus with validate_promql_query before returning it.

TODO: Add the example implementation.
