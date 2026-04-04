package httpapi

import (
	"fmt"
	"strings"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	hrdomain "github.com/BetterAndBetterII/openase/internal/domain/hradvisor"
)

type hrActivityTrendAccumulator struct {
	mergedPRCount  int
	docUpdateCount int
	failureCount   int
	lastSeen       time.Time
}

func parseHRActivityTrends(items []catalogdomain.ActivityEvent) []hrdomain.ActivityTrendContext {
	if len(items) == 0 {
		return nil
	}

	var accumulator hrActivityTrendAccumulator
	for _, item := range items {
		if item.CreatedAt.After(accumulator.lastSeen) {
			accumulator.lastSeen = item.CreatedAt
		}
		if isMergeLikeActivity(item) {
			accumulator.mergedPRCount++
		}
		if isDocumentationUpdateActivity(item) {
			accumulator.docUpdateCount++
		}
		if isFailureActivity(item) {
			accumulator.failureCount++
		}
	}

	trends := make([]hrdomain.ActivityTrendContext, 0, 2)
	if docDriftCount := accumulator.mergedPRCount - accumulator.docUpdateCount; accumulator.mergedPRCount >= 2 && docDriftCount > 0 {
		trends = append(trends, hrdomain.ActivityTrendContext{
			Kind:  hrdomain.ActivityTrendDocumentationDrift,
			Count: docDriftCount,
			Evidence: []string{
				fmt.Sprintf("Recent merge-like activity events: %d.", accumulator.mergedPRCount),
				fmt.Sprintf("Recent documentation update events: %d.", accumulator.docUpdateCount),
			},
			LastSeen: accumulator.lastSeen,
		})
	}
	if accumulator.failureCount >= 3 {
		trends = append(trends, hrdomain.ActivityTrendContext{
			Kind:  hrdomain.ActivityTrendFailureBurst,
			Count: accumulator.failureCount,
			Evidence: []string{
				fmt.Sprintf("Recent failure-like activity events: %d.", accumulator.failureCount),
			},
			LastSeen: accumulator.lastSeen,
		})
	}

	return trends
}

func isMergeLikeActivity(item catalogdomain.ActivityEvent) bool {
	eventType := normalizedActivityText(item.EventType.String())
	message := normalizedActivityText(item.Message)
	return strings.Contains(eventType, "pr.merged") ||
		strings.Contains(eventType, "pull_request.merged") ||
		strings.Contains(eventType, "merged") ||
		strings.Contains(message, "merged pull request") ||
		strings.Contains(message, "pull request merged") ||
		strings.Contains(message, "merged pr") ||
		strings.Contains(message, "pr merged")
}

func isDocumentationUpdateActivity(item catalogdomain.ActivityEvent) bool {
	eventType := normalizedActivityText(item.EventType.String())
	message := normalizedActivityText(item.Message)
	if strings.Contains(eventType, "docs.") || strings.Contains(eventType, "doc.") || strings.Contains(eventType, "documentation") {
		return true
	}
	if strings.Contains(message, "without docs") || strings.Contains(message, "without documentation") ||
		strings.Contains(message, "no docs") || strings.Contains(message, "no documentation") {
		return false
	}

	hasDocKeyword := false
	for _, keyword := range []string{
		"docs",
		"documentation",
		"readme",
		"runbook",
		"guide",
		"manual",
		"changelog",
	} {
		if strings.Contains(message, keyword) {
			hasDocKeyword = true
			break
		}
	}
	if !hasDocKeyword {
		return false
	}
	for _, keyword := range []string{"update", "updated", "refresh", "refreshed", "publish", "published", "write", "wrote", "edit", "edited"} {
		if strings.Contains(message, keyword) {
			return true
		}
	}
	return false
}

func isFailureActivity(item catalogdomain.ActivityEvent) bool {
	eventType := normalizedActivityText(item.EventType.String())
	return strings.Contains(eventType, "failed") || strings.Contains(eventType, "error")
}

func normalizedActivityText(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}
