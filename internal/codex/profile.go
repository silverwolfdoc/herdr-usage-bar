/**
 * Codex multi-account profile model.
 *
 * A profile is one CODEX_HOME-scoped account. Codex writes its auth and its
 * rollout jsonl sessions under that home, so two accounts never collide on
 * disk. Absence of any configured profile synthesizes today's single implicit
 * "codex" profile, whose home is byte-identical to the historical
 * CODEX_HOME-else-~/.codex resolution.
 *
 * Unlike Claude's config dir, CODEX_HOME is the same value on both the write
 * side (a session's own env) and the read side (the panel process inherits the
 * env Herdr was started with), so the implicit profile may read it directly.
 * Multi-account isolation is still opt-in via [[codex.profiles]], because one
 * global env var cannot describe several accounts.
 */
package codex

import "path/filepath"

// DefaultProfileID is the provider id used for the single implicit profile.
const DefaultProfileID = "codex"

// DefaultProfileLabel is the display label for the single implicit profile.
const DefaultProfileLabel = "Codex"

// ProfileSpec is one unresolved [[codex.profiles]] config entry.
type ProfileSpec struct {
	ID        string
	Label     string
	CodexHome string
}

// CodexProfile is a resolved profile with concrete absolute paths.
type CodexProfile struct {
	ID           string
	Label        string
	CodexHome    string
	SessionsRoot string // rollout jsonl root for this account
	AuthPath     string // auth.json for this account
}

func derivedSessionsRoot(codexHome string) string {
	return filepath.Join(codexHome, "sessions")
}

func derivedAuthPath(codexHome string) string {
	return filepath.Join(codexHome, "auth.json")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// synthesizeDefaultProfile builds the single implicit "codex" profile, keeping
// the historical CODEX_HOME-else-~/.codex resolution exactly.
func synthesizeDefaultProfile(env map[string]string, home string) CodexProfile {
	codexHome := firstNonEmpty(env["CODEX_HOME"], filepath.Join(home, ".codex"))
	return newProfile(DefaultProfileID, DefaultProfileLabel, codexHome)
}

func newProfile(id, label, codexHome string) CodexProfile {
	return CodexProfile{
		ID:           id,
		Label:        label,
		CodexHome:    codexHome,
		SessionsRoot: derivedSessionsRoot(codexHome),
		AuthPath:     derivedAuthPath(codexHome),
	}
}

// ResolveProfiles turns config entries into concrete profiles.
//
//   - No specs -> one synthesized default "codex" profile (backward compat).
//   - Otherwise each valid spec becomes a profile. Entries missing id or
//     codex_home are skipped, as are duplicate ids and duplicate homes (first
//     wins), so malformed config degrades safely rather than collides.
//
// CODEX_HOME is deliberately ignored once profiles are configured: a single
// global env var cannot be attributed to one of several accounts, and letting
// it win would make the panel disagree with itself depending on which shell
// started Herdr.
func ResolveProfiles(specs []ProfileSpec, env map[string]string, home string) []CodexProfile {
	if len(specs) == 0 {
		return []CodexProfile{synthesizeDefaultProfile(env, home)}
	}
	out := make([]CodexProfile, 0, len(specs))
	seenID := map[string]bool{}
	seenHome := map[string]bool{}
	for _, spec := range specs {
		if spec.ID == "" || spec.CodexHome == "" {
			continue
		}
		if seenID[spec.ID] || seenHome[spec.CodexHome] {
			continue
		}
		seenID[spec.ID] = true
		seenHome[spec.CodexHome] = true
		out = append(out, newProfile(spec.ID, firstNonEmpty(spec.Label, spec.ID), spec.CodexHome))
	}
	if len(out) == 0 {
		return []CodexProfile{synthesizeDefaultProfile(env, home)}
	}
	return out
}

// ProfileForHome picks the profile owning codexHome.
//
//   - Single profile: always returns it (ok=true) — the single-profile fallback.
//   - Multiple profiles: returns the home match, or ok=false when none match,
//     so a caller can skip rather than misattribute the account.
func ProfileForHome(profiles []CodexProfile, codexHome string) (CodexProfile, bool) {
	if len(profiles) == 1 {
		return profiles[0], true
	}
	for _, p := range profiles {
		if p.CodexHome == codexHome {
			return p, true
		}
	}
	return CodexProfile{}, false
}

// SessionsRoots returns every configured account's rollout root, in config
// order. Used by the per-pane resolver, which must find a pane's rollout
// without knowing which account that pane runs under.
func SessionsRoots(profiles []CodexProfile) []string {
	out := make([]string, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, p.SessionsRoot)
	}
	return out
}

// IsDefaultProfile reports whether p is the lone implicit default profile,
// used to decide whether notification titles get a label prefix.
func IsDefaultProfile(p CodexProfile) bool {
	return p.ID == DefaultProfileID && p.Label == DefaultProfileLabel
}

// IsCodexProviderID reports whether a provider id belongs to any configured
// Codex profile. Replaces literal `== "codex"` checks now that a profile's id
// may be e.g. "codex-vvs".
func IsCodexProviderID(id string, profiles []CodexProfile) bool {
	for _, p := range profiles {
		if p.ID == id {
			return true
		}
	}
	return false
}
