package codex

import (
	"path/filepath"
	"testing"
)

const home = "/home/u"

func TestResolveProfilesNoSpecsSynthesizesDefaultFromEnv(t *testing.T) {
	got := ResolveProfiles(nil, map[string]string{"CODEX_HOME": "/custom/.codex"}, home)
	if len(got) != 1 {
		t.Fatalf("want 1 profile, got %d", len(got))
	}
	if got[0].ID != DefaultProfileID || got[0].Label != DefaultProfileLabel {
		t.Fatalf("want default id/label, got %q/%q", got[0].ID, got[0].Label)
	}
	if got[0].CodexHome != "/custom/.codex" {
		t.Fatalf("CODEX_HOME must win for the implicit profile, got %q", got[0].CodexHome)
	}
	if want := filepath.Join("/custom/.codex", "sessions"); got[0].SessionsRoot != want {
		t.Fatalf("SessionsRoot = %q, want %q", got[0].SessionsRoot, want)
	}
}

func TestResolveProfilesNoSpecsNoEnvUsesHomeDotCodex(t *testing.T) {
	got := ResolveProfiles(nil, map[string]string{}, home)
	if want := filepath.Join(home, ".codex"); got[0].CodexHome != want {
		t.Fatalf("CodexHome = %q, want %q", got[0].CodexHome, want)
	}
}

func TestResolveProfilesUsesSpecsAndIgnoresEnv(t *testing.T) {
	specs := []ProfileSpec{
		{ID: "almir", Label: "Codex almir", CodexHome: "/home/u/.codex-almir"},
		{ID: "vvs", CodexHome: "/home/u/.codex-vvs"},
	}
	got := ResolveProfiles(specs, map[string]string{"CODEX_HOME": "/ignored"}, home)
	if len(got) != 2 {
		t.Fatalf("want 2 profiles, got %d", len(got))
	}
	if got[0].Label != "Codex almir" {
		t.Fatalf("explicit label lost: %q", got[0].Label)
	}
	if got[1].Label != "vvs" {
		t.Fatalf("missing label must fall back to id, got %q", got[1].Label)
	}
	for _, p := range got {
		if p.CodexHome == "/ignored" {
			t.Fatal("CODEX_HOME must not leak into configured profiles")
		}
	}
}

func TestResolveProfilesSkipsIncompleteAndDuplicateEntries(t *testing.T) {
	specs := []ProfileSpec{
		{ID: "", CodexHome: "/a"},
		{ID: "b", CodexHome: ""},
		{ID: "ok", CodexHome: "/ok"},
		{ID: "ok", CodexHome: "/other"},
		{ID: "other", CodexHome: "/ok"},
	}
	got := ResolveProfiles(specs, nil, home)
	if len(got) != 1 || got[0].ID != "ok" {
		t.Fatalf("want only the first valid unique entry, got %+v", got)
	}
}

func TestResolveProfilesAllInvalidFallsBackToDefault(t *testing.T) {
	got := ResolveProfiles([]ProfileSpec{{ID: "x"}}, map[string]string{}, home)
	if len(got) != 1 || got[0].ID != DefaultProfileID {
		t.Fatalf("want synthesized default, got %+v", got)
	}
}

func TestProfileForHomeMatchesAndFallsBack(t *testing.T) {
	ps := ResolveProfiles([]ProfileSpec{
		{ID: "almir", CodexHome: "/home/u/.codex-almir"},
		{ID: "vvs", CodexHome: "/home/u/.codex-vvs"},
	}, nil, home)

	got, ok := ProfileForHome(ps, "/home/u/.codex-vvs")
	if !ok || got.ID != "vvs" {
		t.Fatalf("want vvs match, got %+v ok=%v", got, ok)
	}
	if _, ok := ProfileForHome(ps, "/home/u/.codex-nope"); ok {
		t.Fatal("unknown home must not match when several profiles exist")
	}
	single := ResolveProfiles(nil, map[string]string{}, home)
	if _, ok := ProfileForHome(single, "/anything"); !ok {
		t.Fatal("single profile must always match (backward compat)")
	}
}

func TestIsDefaultProfile(t *testing.T) {
	if !IsDefaultProfile(ResolveProfiles(nil, map[string]string{}, home)[0]) {
		t.Fatal("synthesized profile must report as default")
	}
	if IsDefaultProfile(ResolveProfiles([]ProfileSpec{{ID: "almir", CodexHome: "/x"}}, nil, home)[0]) {
		t.Fatal("configured profile must not report as default")
	}
}
