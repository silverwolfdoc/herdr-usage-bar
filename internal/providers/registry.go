/**
 * UsageProvider registry for the supported agents.
 * To add a new agent, register it here and add a providers/<agent>/ directory.
 */
package providers

import (
	"github.com/silverwolfdoc/herdr-usage-bar/internal/provider"
	"github.com/silverwolfdoc/herdr-usage-bar/internal/providers/claude"
	"github.com/silverwolfdoc/herdr-usage-bar/internal/providers/codex"
	"github.com/silverwolfdoc/herdr-usage-bar/internal/providers/grok"
	"github.com/silverwolfdoc/herdr-usage-bar/internal/providers/omp"
	"github.com/silverwolfdoc/herdr-usage-bar/internal/providers/opencode"
)

// All registered providers.
var All = []provider.UsageProvider{
	claude.Provider,
	codex.Provider,
	grok.Provider,
	omp.Provider,
	omp.PiProvider,
	opencode.Provider,
}

// FindProvider returns the provider for agentId, or nil when unregistered.
func FindProvider(agentID string) provider.UsageProvider {
	for _, p := range All {
		if p.AgentID() == agentID {
			return p
		}
	}
	return nil
}
