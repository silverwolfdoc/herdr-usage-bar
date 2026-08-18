/**
 * Tests for resolving a pane's rollout across several configured CODEX_HOMEs.
 */
package codex

import (
	"path/filepath"
	"testing"
	"time"
)

// restoreRoots resets the package-level override so tests stay independent.
func restoreRoots(t *testing.T) {
	t.Helper()
	prev := SessionsRootsOverride
	t.Cleanup(func() { SessionsRootsOverride = prev })
}

// A pane started under the SECOND configured account must still be found. With
// a single implicit home its rollout is invisible, so the pane shows no context
// meter at all.
func TestFindSessionFileAcrossConfiguredHomes(t *testing.T) {
	restoreRoots(t)
	root := t.TempDir()
	almir := filepath.Join(root, ".codex-almir")
	vvs := filepath.Join(root, ".codex-vvs")
	t.Setenv("CODEX_HOME", almir) // the implicit single home
	now := time.Now()
	writeRollout(t, almir, "sess-almir", "2026/07/12", now)
	want := writeRollout(t, vvs, "sess-vvs", "2026/07/12", now)

	if got := FindSessionFile("sess-vvs"); got != "" {
		t.Fatalf("precondition: second home must be invisible before the override, got %q", got)
	}

	SessionsRootsOverride = []string{
		filepath.Join(almir, "sessions"),
		filepath.Join(vvs, "sessions"),
	}
	if got := FindSessionFile("sess-vvs"); got != want {
		t.Fatalf("FindSessionFile = %q, want %q", got, want)
	}
	if got := FindSessionFile("sess-almir"); got == "" {
		t.Fatal("first home must still resolve once the override is set")
	}
}

// Newest-first ordering is the callers' contract; merging two roots must not
// break it.
func TestRolloutCandidatesMergeHomesNewestFirst(t *testing.T) {
	restoreRoots(t)
	root := t.TempDir()
	a := filepath.Join(root, ".codex-a")
	b := filepath.Join(root, ".codex-b")
	now := time.Now()
	writeRollout(t, a, "older", "2026/07/12", now.Add(-2*time.Hour))
	newest := writeRollout(t, b, "newer", "2026/07/12", now)

	SessionsRootsOverride = []string{filepath.Join(a, "sessions"), filepath.Join(b, "sessions")}
	got := listRolloutCandidates()
	if len(got) != 2 {
		t.Fatalf("want both homes' rollouts, got %d", len(got))
	}
	if got[0].path != newest {
		t.Fatalf("newest-first broken across homes: %q first", got[0].path)
	}
}

// No override configured = today's behaviour, exactly.
func TestSessionsRootsFallsBackToSingleHome(t *testing.T) {
	restoreRoots(t)
	SessionsRootsOverride = nil
	home := withTempCodexHome(t)
	roots := sessionsRoots()
	if len(roots) != 1 || roots[0] != filepath.Join(home, "sessions") {
		t.Fatalf("want the single implicit root, got %+v", roots)
	}
}
