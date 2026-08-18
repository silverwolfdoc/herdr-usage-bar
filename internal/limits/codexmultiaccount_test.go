package limits

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/silverwolfdoc/herdr-usage-bar/internal/codex"
)

// Two configured accounts must report their OWN numbers. Before multi-account
// support both rows read whichever session turned most recently, so a drained
// seat could show a healthy one's percentage.
func TestCollectCodexLimitsForProfileIsScopedToItsOwnHome(t *testing.T) {
	root := t.TempDir()
	almir := filepath.Join(root, ".codex-almir")
	vvs := filepath.Join(root, ".codex-vvs")
	now := time.Now()
	writeRollout(t, almir, "almir-sess", "/repo/a", 98, now) // 2% left
	writeRollout(t, vvs, "vvs-sess", "/repo/b", 100, now)    // 0% left

	profiles := codex.ResolveProfiles([]codex.ProfileSpec{
		{ID: "almir", Label: "Codex almir", CodexHome: almir},
		{ID: "vvs", Label: "Codex vvs", CodexHome: vvs},
	}, nil, root)

	got := map[string]float64{}
	for _, p := range profiles {
		pl := CollectCodexLimitsForProfile(p, 1000)
		if pl.ProviderID != p.ID || pl.Label != p.Label {
			t.Fatalf("profile identity lost: %+v", pl)
		}
		if pl.Primary == nil {
			t.Fatalf("%s: no primary window (note=%v)", p.ID, pl.Note)
		}
		got[p.ID] = pl.Primary.UsedPercentage
	}
	if got["almir"] != 98 || got["vvs"] != 100 {
		t.Fatalf("accounts crossed over: %+v", got)
	}
}

func TestCollectCodexLimitsForProfileNotesItsOwnRootWhenEmpty(t *testing.T) {
	root := t.TempDir()
	p := codex.ResolveProfiles([]codex.ProfileSpec{{ID: "vvs", CodexHome: filepath.Join(root, ".codex-vvs")}}, nil, root)[0]
	pl := CollectCodexLimitsForProfile(p, 1000)
	if pl.Note == nil || !containsStr(*pl.Note, ".codex-vvs") {
		t.Fatalf("note must name the account's own root, got %v", pl.Note)
	}
}

// Display order: every configured Codex account renders after Claude and before
// OpenCode, in config order.
func TestCollectAllProviderLimitsKeepsCodexAccountOrder(t *testing.T) {
	mk := func(id string) CodexProfileCollector {
		return CodexProfileCollector{ID: id, Label: id, Collector: func(_ *string, now int64) ProviderLimits {
			return ProviderLimits{ProviderID: id, Label: id, Source: "test", FetchedAtMs: now}
		}}
	}
	got := CollectAllProviderLimits(nil, 1, CollectOptions{
		Claude: []ClaudeProfileCollector{{ID: "claude", Label: "Claude", Collector: func(_ *string, now int64) ProviderLimits {
			return ProviderLimits{ProviderID: "claude", FetchedAtMs: now}
		}}},
		Codex: []CodexProfileCollector{mk("almir"), mk("vvs")},
	})
	want := []string{"claude", "almir", "vvs", "opencode", "grok"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d (%+v)", len(got), len(want), got)
	}
	for i, id := range want {
		if got[i].ProviderID != id {
			t.Fatalf("[%d] = %q want %q", i, got[i].ProviderID, id)
		}
	}
}

func TestCollectAllProviderLimitsOnlyFiltersPerCodexAccount(t *testing.T) {
	called := map[string]bool{}
	mk := func(id string) CodexProfileCollector {
		return CodexProfileCollector{ID: id, Label: id, Collector: func(_ *string, now int64) ProviderLimits {
			called[id] = true
			return ProviderLimits{ProviderID: id, FetchedAtMs: now}
		}}
	}
	got := CollectAllProviderLimits(nil, 1, CollectOptions{
		Codex: []CodexProfileCollector{mk("almir"), mk("vvs")},
		Only:  map[string]bool{"vvs": true},
	})
	if called["almir"] {
		t.Fatal("filtered-out account's collector ran")
	}
	if len(got) != 1 || got[0].ProviderID != "vvs" {
		t.Fatalf("got %+v", got)
	}
}

// A Codex pane must activate EVERY configured Codex account id. Without this
// the panel's Only-filter matches no id and the whole Codex block disappears —
// measured on hardware 2026-08-18 before the fix.
func TestActiveProviderSetActivatesAllConfiguredCodexAccounts(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERDR_PLUGIN_CONFIG_DIR", dir)
	writeRaw(t, filepath.Join(dir, "config.toml"), `
[[codex.profiles]]
id = "codex-almir"
codex_home = "/home/u/.codex-almir"

[[codex.profiles]]
id = "codex-vvs"
codex_home = "/home/u/.codex-vvs"
`)
	got := ActiveProviderSet([]OpenPaneSnapshot{{Agent: "codex"}})
	for _, id := range []string{"codex-almir", "codex-vvs"} {
		if !got[id] {
			t.Fatalf("%q not activated by a codex pane: %+v", id, got)
		}
	}
}

func TestActiveProviderSetKeepsLiteralCodexWhenUnconfigured(t *testing.T) {
	isolatePluginConfig(t)
	got := ActiveProviderSet([]OpenPaneSnapshot{{Agent: "codex"}})
	if !got["codex"] {
		t.Fatalf("unconfigured setup must still activate the literal id: %+v", got)
	}
}

// The share denominator must be that account's own traffic; summing all homes
// would shrink every pane's reported share.
func TestTotalTokensForProviderScopesToTheAccountsOwnRoot(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, ".codex-a")
	b := filepath.Join(root, ".codex-b")
	now := time.Now()
	writeRollout(t, a, "a1", "/repo/a", 5, now)
	writeRollout(t, b, "b1", "/repo/b", 5, now)
	profiles := codex.ResolveProfiles([]codex.ProfileSpec{
		{ID: "codex-a", CodexHome: a}, {ID: "codex-b", CodexHome: b},
	}, nil, root)

	start := now.Add(-time.Hour).UnixMilli()
	end := now.Add(time.Hour).UnixMilli()
	only := totalTokensForProviderWith2(nil, profiles, "codex-a", start, end)
	both := codexTotalIn(filepath.Join(a, "sessions"), start, end) + codexTotalIn(filepath.Join(b, "sessions"), start, end)
	if only == both && both != 0 {
		t.Fatalf("account total must not include the other account: %v vs %v", only, both)
	}
}

func TestBillingProviderFilterCoversConfiguredCodexAccounts(t *testing.T) {
	got := BillingProviderFilter(nil, false, BillingDeps{
		ClaudeProfileIDs: []string{"claude"},
		CodexProfileIDs:  []string{"codex-almir", "codex-vvs"},
	})
	for _, id := range []string{"codex-almir", "codex-vvs"} {
		if _, ok := got[id]; !ok {
			t.Fatalf("%q missing from the billing universe: %+v", id, got)
		}
	}
}
