/**
 * Maps Claude-specific TranscriptUsage to the core ContextUsage.
 */
package claude

import (
	claudemodels "github.com/silverwolfdoc/herdr-usage-bar/internal/claude"
	"github.com/silverwolfdoc/herdr-usage-bar/internal/core"
)

// ToContextUsage maps TranscriptUsage to core.ContextUsage.
func ToContextUsage(usage TranscriptUsage) core.ContextUsage {
	contextTokens := ContextTokensOf(usage)
	window := claudemodels.ContextWindowFor(usage.Model)
	if window == nil {
		return core.ContextUsage{ContextTokens: contextTokens}
	}
	w := *window
	return core.ContextUsage{ContextTokens: contextTokens, WindowTokens: &w}
}
