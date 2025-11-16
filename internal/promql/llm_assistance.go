package promql

import (
	"context"
	"fmt"
	"strings"
)

// llmQueryEnhancer provides LLM-assisted query enhancement
type llmQueryEnhancer struct {
	// In a real implementation, this would contain an LLM client
	// For now, we'll use rule-based enhancement with intelligent heuristics
}

// newLLMQueryEnhancer creates a new LLM query enhancer
func newLLMQueryEnhancer() *llmQueryEnhancer {
	return &llmQueryEnhancer{}
}

// enhanceQueries enhances query suggestions using LLM-like intelligence
func (e *llmQueryEnhancer) enhanceQueries(ctx context.Context, metricInfo *MetricInfo, suggestions []QuerySuggestion) []QuerySuggestion {
	enhanced := make([]QuerySuggestion, 0, len(suggestions))

	for _, suggestion := range suggestions {
		enhancedQuery := e.enhanceQuery(metricInfo, suggestion)
		enhanced = append(enhanced, enhancedQuery)
	}

	contextualQueries := e.generateContextualQueries(metricInfo)
	enhanced = append(enhanced, contextualQueries...)

	return enhanced
}

// enhanceQuery improves a single query suggestion with additional context
func (e *llmQueryEnhancer) enhanceQuery(metricInfo *MetricInfo, suggestion QuerySuggestion) QuerySuggestion {
	enhanced := suggestion

	enhanced.Description = e.enhanceDescription(metricInfo, suggestion)

	enhanced.Query = e.optimizeQuery(metricInfo, suggestion.Query)

	enhanced.VisualizationType = e.suggestVisualizationType(metricInfo, suggestion)

	return enhanced
}

// enhanceDescription improves query descriptions with contextual information
func (e *llmQueryEnhancer) enhanceDescription(metricInfo *MetricInfo, suggestion QuerySuggestion) string {
	baseName := metricInfo.Name
	baseDesc := suggestion.Description

	if strings.Contains(baseName, "http") {
		if strings.Contains(suggestion.Query, "rate(") {
			return fmt.Sprintf("HTTP %s", strings.ToLower(baseDesc))
		}
	}

	if strings.Contains(baseName, "error") || strings.Contains(baseName, "fail") {
		if strings.Contains(suggestion.Query, "rate(") {
			return fmt.Sprintf("Error %s", strings.ToLower(baseDesc))
		}
	}

	if strings.Contains(baseName, "memory") || strings.Contains(baseName, "cpu") {
		return fmt.Sprintf("Resource Usage: %s", baseDesc)
	}

	if strings.Contains(baseName, "latency") || strings.Contains(baseName, "duration") {
		return fmt.Sprintf("Performance: %s", baseDesc)
	}

	if strings.Contains(suggestion.Query, "histogram_quantile") {
		percentile := extractPercentile(suggestion.Query)
		if percentile != "" {
			return fmt.Sprintf("%s percentile (%s of requests are faster)", percentile, percentile)
		}
	}

	if strings.Contains(suggestion.Query, "increase(") {
		return fmt.Sprintf("Total %s", strings.ToLower(baseDesc))
	}

	return baseDesc
}

// optimizeQuery improves query performance and accuracy
func (e *llmQueryEnhancer) optimizeQuery(metricInfo *MetricInfo, query string) string {
	optimized := query

	if strings.Contains(query, "rate(") && strings.Contains(query, "[5m]") {
		if strings.Contains(metricInfo.Name, "request") || strings.Contains(metricInfo.Name, "http") {
			optimized = strings.ReplaceAll(optimized, "[5m]", "[2m]")
		}
	}

	if strings.Contains(metricInfo.Name, "error") && strings.Contains(query, "rate(") {
		if !strings.Contains(query, "sum") {
			optimized = strings.ReplaceAll(optimized, "rate(", "rate(")
		}
	}

	if strings.Contains(query, "histogram_quantile") {
		if !strings.Contains(query, "sum(rate(") && !strings.Contains(query, "sum by") {
			metricName := extractMetricNameFromHistogramQuery(query)
			if metricName != "" {
				optimized = strings.ReplaceAll(optimized,
					fmt.Sprintf("rate(%s_bucket[", metricName),
					fmt.Sprintf("sum(rate(%s_bucket[", metricName))
				if strings.Count(optimized, "sum(") == 1 {
					optimized = strings.ReplaceAll(optimized, "]))", "])) by (le)")
				}
			}
		}
	}

	return optimized
}

// suggestVisualizationType recommends the best visualization type
func (e *llmQueryEnhancer) suggestVisualizationType(metricInfo *MetricInfo, suggestion QuerySuggestion) string {
	vizType := suggestion.VisualizationType

	if strings.Contains(suggestion.Query, "histogram_quantile") {
		return "timeseries"
	}

	if strings.Contains(suggestion.Query, "avg(") && !strings.Contains(suggestion.Query, "over_time") {
		return "stat"
	}

	if strings.Contains(suggestion.Query, "max(") || strings.Contains(suggestion.Query, "min(") {
		return "stat"
	}

	if strings.Contains(metricInfo.Name, "ratio") || strings.Contains(metricInfo.Name, "percent") {
		return "gauge"
	}

	return vizType
}

// generateContextualQueries creates additional relevant queries based on context
func (e *llmQueryEnhancer) generateContextualQueries(metricInfo *MetricInfo) []QuerySuggestion {
	var contextual []QuerySuggestion
	metricName := metricInfo.Name

	if strings.Contains(metricName, "http_request") || strings.Contains(metricName, "request") {
		if metricInfo.Type == MetricTypeCounter {
			contextual = append(contextual, QuerySuggestion{
				Query:             fmt.Sprintf("rate(%s{status=~\"5..\"}[5m]) / rate(%s[5m])", metricName, metricName),
				Description:       "Error rate (5xx responses)",
				VisualizationType: "timeseries",
				YAxisLabel:        "error ratio",
			})

			contextual = append(contextual, QuerySuggestion{
				Query:             fmt.Sprintf("rate(%s{status=~\"2..\"}[5m]) / rate(%s[5m])", metricName, metricName),
				Description:       "Success rate (2xx responses)",
				VisualizationType: "stat",
				YAxisLabel:        "success ratio",
			})
		}
	}

	if metricInfo.Type == MetricTypeCounter && (strings.Contains(metricName, "error") || strings.Contains(metricName, "fail")) {
		contextual = append(contextual, QuerySuggestion{
			Query:             fmt.Sprintf("increase(%s[1h]) > 10", metricName),
			Description:       "High error count alert (>10/hour)",
			VisualizationType: "table",
			YAxisLabel:        "count",
		})
	}

	if strings.Contains(metricName, "cpu") && metricInfo.Type == MetricTypeGauge {
		contextual = append(contextual, QuerySuggestion{
			Query:             fmt.Sprintf("(%s > 80)", metricName),
			Description:       "High CPU usage alert (>80%)",
			VisualizationType: "table",
			YAxisLabel:        "percent",
		})
	}

	if strings.Contains(metricName, "memory") && metricInfo.Type == MetricTypeGauge {
		contextual = append(contextual, QuerySuggestion{
			Query:             fmt.Sprintf("(%s / 1024 / 1024 / 1024)", metricName),
			Description:       "Memory usage in GB",
			VisualizationType: "timeseries",
			YAxisLabel:        "GB",
		})
	}

	return contextual
}

// Helper functions

func extractPercentile(query string) string {
	if strings.Contains(query, "0.50") {
		return "50th"
	}
	if strings.Contains(query, "0.95") {
		return "95th"
	}
	if strings.Contains(query, "0.99") {
		return "99th"
	}
	if strings.Contains(query, "0.90") {
		return "90th"
	}
	return ""
}

func extractMetricNameFromHistogramQuery(query string) string {
	if strings.Contains(query, "_bucket") {
		bucketIndex := strings.Index(query, "_bucket")
		if bucketIndex == -1 {
			return ""
		}

		beforeBucket := query[:bucketIndex]

		words := strings.FieldsFunc(beforeBucket, func(r rune) bool {
			return r == ' ' || r == '(' || r == ',' || r == ')'
		})

		if len(words) == 0 {
			return ""
		}

		lastWord := words[len(words)-1]

		lastWord = strings.Trim(lastWord, "()[], ")

		return lastWord
	}
	return ""
}
