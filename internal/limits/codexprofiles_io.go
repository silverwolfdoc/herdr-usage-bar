/**
 * Resolves the configured Codex profiles from the plugin config, shared by the
 * read-side collectors (panel, sidebar) and the per-pane rollout resolver.
 */
package limits

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"

	"github.com/silverwolfdoc/herdr-usage-bar/internal/codex"
	"github.com/silverwolfdoc/herdr-usage-bar/internal/setup"
)

// ResolvedCodexProfiles resolves the configured Codex profiles (synthesizing
// the single implicit default when none are configured) from process env.
func ResolvedCodexProfiles() []codex.CodexProfile {
	return setup.ResolveCodexProfiles(processEnvMap())
}

// CodexSessionsRoots is every configured account's rollout root, in config
// order, for the per-pane resolver.
func CodexSessionsRoots() []string {
	return codex.SessionsRoots(ResolvedCodexProfiles())
}

// CodexAccountEmailFromAuthPath reads the logged-in account's email out of a
// Codex auth.json, so several configured accounts stay distinguishable in the
// panel even when their labels are generic.
//
// The email lives in the OIDC id_token's payload; the token is only decoded,
// never verified — this is a display label, not a credential, and the file is
// already the account's own private auth state.
func CodexAccountEmailFromAuthPath(authPath string) (string, bool) {
	raw, err := os.ReadFile(authPath)
	if err != nil {
		return "", false
	}
	var auth struct {
		Tokens *struct {
			IDToken string `json:"id_token"`
		} `json:"tokens"`
	}
	if err := json.Unmarshal(raw, &auth); err != nil || auth.Tokens == nil {
		return "", false
	}
	parts := strings.Split(auth.Tokens.IDToken, ".")
	if len(parts) < 2 {
		return "", false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", false
	}
	var claims struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Email == "" {
		return "", false
	}
	return claims.Email, true
}

// applyCodexProfileGrouping nests pl under the shared "Codex" heading when
// multiProfile is true, so 2+ configured accounts render as one group instead
// of N separate top-level blocks. AccountLabel carries the account's real
// logged-in email, falling back to the profile label when auth.json can't be
// read (a logged-out account still gets a distinguishable row).
func applyCodexProfileGrouping(pl ProviderLimits, p codex.CodexProfile, multiProfile bool) ProviderLimits {
	if !multiProfile {
		return pl
	}
	pl.GroupLabel = "Codex"
	if email, ok := CodexAccountEmailFromAuthPath(p.AuthPath); ok {
		pl.AccountLabel = email
	} else {
		pl.AccountLabel = pl.Label
	}
	return pl
}
