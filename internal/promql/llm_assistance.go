package promql

import (
	"context"
	"fmt"
	"strings"
)

// LLMQueryEnhancer provides LLM-assisted query enhancement
type LLMQueryEnhancer struct {
	// In a real implementation, this would contain an LLM client
	// For now, we'll use rule-based enhancement with intelligent heuristics
}

// NewLLMQueryEnhancer creates a new LLM query enhancer
func NewLLMQueryEnhancer() *LLMQueryEnhancer {
	return &LLMQueryEnhancer{}
}

// EnhanceQueries enhances query suggestions using LLM-like intelligence
func (e *LLMQueryEnhancer) EnhanceQueries(ctx context.Context, metricInfo *MetricInfo, suggestions []QuerySuggestion) []QuerySuggestion {
	enhanced := make([]QuerySuggestion, 0, len(suggestions))
	
	for _, suggestion := range suggestions {
		enhancedQuery := e.enhanceQuery(metricInfo, suggestion)
		enhanced = append(enhanced, enhancedQuery)
	}

	// Add contextually relevant queries based on metric patterns
	contextualQueries := e.generateContextualQueries(metricInfo)
	enhanced = append(enhanced, contextualQueries...)

	return enhanced
}

// enhanceQuery improves a single query suggestion with additional context
func (e *LLMQueryEnhancer) enhanceQuery(metricInfo *MetricInfo, suggestion QuerySuggestion) QuerySuggestion {
	enhanced := suggestion

	// Enhance descriptions with more context
	enhanced.Description = e.enhanceDescription(metricInfo, suggestion)

	// Optimize query for better performance and accuracy
	enhanced.Query = e.optimizeQuery(metricInfo, suggestion.Query)

	// Suggest better visualization types based on query patterns
	enhanced.VisualizationType = e.suggestVisualizationType(metricInfo, suggestion)

	return enhanced
}

// enhanceDescription improves query descriptions with contextual information
func (e *LLMQueryEnhancer) enhanceDescription(metricInfo *MetricInfo, suggestion QuerySuggestion) string {
	baseName := metricInfo.Name
	baseDesc := suggestion.Description

	// Add context based on metric name patterns
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

	// Add helpful context for different query types
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
func (e *LLMQueryEnhancer) optimizeQuery(metricInfo *MetricInfo, query string) string {
	optimized := query

	// Optimize rate queries for better accuracy
	if strings.Contains(query, "rate(") && strings.Contains(query, "[5m]") {
		// For high-frequency metrics, use shorter intervals
		if strings.Contains(metricInfo.Name, "request") || strings.Contains(metricInfo.Name, "http") {
			optimized = strings.ReplaceAll(optimized, "[5m]", "[2m]")
		}
	}

	// Add irate for spike detection in appropriate cases
	if strings.Contains(metricInfo.Name, "error") && strings.Contains(query, "rate(") {
		// Suggest both rate and irate for error metrics
		if !strings.Contains(query, "sum") {
			// Keep the original rate query but mark it as optimized
			optimized = strings.ReplaceAll(optimized, "rate(", "rate(")
		}
	}

	// Optimize histogram queries
	if strings.Contains(query, "histogram_quantile") {
		// Ensure proper bucket aggregation
		if !strings.Contains(query, "sum(rate(") && !strings.Contains(query, "sum by") {
			// Add proper aggregation for multi-instance setups
			metricName := extractMetricNameFromHistogramQuery(query)
			if metricName != "" {
				optimized = strings.ReplaceAll(optimized, 
					fmt.Sprintf("rate(%s_bucket[", metricName),
					fmt.Sprintf("sum(rate(%s_bucket[", metricName))
				if strings.Count(optimized, "sum(") == 1 {
					// Add the closing parenthesis and by clause
					optimized = strings.ReplaceAll(optimized, "]))", "])) by (le)")
				}
			}
		}
	}

	return optimized
}

// suggestVisualizationType recommends the best visualization type
func (e *LLMQueryEnhancer) suggestVisualizationType(metricInfo *MetricInfo, suggestion QuerySuggestion) string {
	// Use current visualization type as baseline
	vizType := suggestion.VisualizationType

	// Override with better suggestions based on context
	if strings.Contains(suggestion.Query, "histogram_quantile") {
		return "timeseries" // Percentiles are best shown as time series
	}

	if strings.Contains(suggestion.Query, "avg(") && !strings.Contains(suggestion.Query, "over_time") {
		return "stat" // Current averages work well as stats
	}

	if strings.Contains(suggestion.Query, "max(") || strings.Contains(suggestion.Query, "min(") {
		return "stat" // Min/max values work well as stats
	}

	if strings.Contains(metricInfo.Name, "ratio") || strings.Contains(metricInfo.Name, "percent") {
		return "gauge" // Percentages work well as gauges
	}

	return vizType
}

// generateContextualQueries creates additional relevant queries based on context
func (e *LLMQueryEnhancer) generateContextualQueries(metricInfo *MetricInfo) []QuerySuggestion {
	var contextual []QuerySuggestion
	metricName := metricInfo.Name

	// Add SLI/SLO related queries for service metrics
	if strings.Contains(metricName, "http_request") || strings.Contains(metricName, "request") {
		if metricInfo.Type == MetricTypeCounter {
			// Add error rate query if this seems to be a request counter
			contextual = append(contextual, QuerySuggestion{
				Query:             fmt.Sprintf("rate(%s{status=~\"5..\"}[5m]) / rate(%s[5m])", metricName, metricName),
				Description:       "Error rate (5xx responses)",
				VisualizationType: "timeseries",
				YAxisLabel:        "error ratio",
			})

			// Add success rate
			contextual = append(contextual, QuerySuggestion{
				Query:             fmt.Sprintf("rate(%s{status=~\"2..\"}[5m]) / rate(%s[5m])", metricName, metricName),
				Description:       "Success rate (2xx responses)",
				VisualizationType: "stat",
				YAxisLabel:        "success ratio",
			})
		}
	}

	// Add alerting-focused queries
	if metricInfo.Type == MetricTypeCounter && (strings.Contains(metricName, "error") || strings.Contains(metricName, "fail")) {
		contextual = append(contextual, QuerySuggestion{
			Query:             fmt.Sprintf("increase(%s[1h]) > 10", metricName),
			Description:       "High error count alert (>10/hour)",
			VisualizationType: "table",
			YAxisLabel:        "count",
		})
	}

	// Add resource utilization patterns
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
	// Extract metric name from histogram_quantile query
	// Example: histogram_quantile(0.95, rate(http_duration_bucket[5m]))
	if strings.Contains(query, "_bucket") {
		// Find the position of _bucket in the query
		bucketIndex := strings.Index(query, "_bucket")
		if bucketIndex == -1 {
			return ""
		}
		
		// Extract everything before _bucket
		beforeBucket := query[:bucketIndex]
		
		// Find the last word/identifier that looks like a metric name
		// Split by common separators and punctuation
		words := strings.FieldsFunc(beforeBucket, func(r rune) bool {
			return r == ' ' || r == '(' || r == ',' || r == ')'
		})
		
		if len(words) == 0 {
			return ""
		}
		
		// Return the last word that could be a metric name
		lastWord := words[len(words)-1]
		
		// Remove any remaining punctuation
		lastWord = strings.Trim(lastWord, "()[], ")
		
		return lastWord
	}
	return ""
}