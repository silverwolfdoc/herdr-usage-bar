/**
 * Tests for plugin config seed/parse
 */
package setup

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParsePluginConfigTOML_Defaults(t *testing.T) {
	cfg := ParsePluginConfigTOML(DefaultPluginConfigTOML(DefaultPluginConfig))
	if !cfg.NotifyEnabled {
		t.Fatal("expected enabled")
	}
	if !reflect.DeepEqual(cfg.RemainingThresholds, []int{50, 20, 10, 5}) {
		t.Fatalf("thresholds=%v", cfg.RemainingThresholds)
	}
}

func TestParsePluginConfigTOML_Custom(t *testing.T) {
	cfg := ParsePluginConfigTOML(`
[notify]
enabled = false
remaining_thresholds = [30, 10]
`)
	if cfg.NotifyEnabled {
		t.Fatal("expected disabled")
	}
	if !reflect.DeepEqual(cfg.RemainingThresholds, []int{30, 10}) {
		t.Fatalf("thresholds=%v", cfg.RemainingThresholds)
	}
}

func TestParsePluginConfigTOML_ClaudeProfiles(t *testing.T) {
	cfg := ParsePluginConfigTOML(`
[[claude.profiles]]
id = "claude"
label = "Claude"
config_dir = "/home/u/.claude"
claude_json_path = "/home/u/.claude.json"

[[claude.profiles]]
id = "claude-secondary"
label = "Claude (secondary)"
config_dir = "/home/u/.claude-m"
`)
	if len(cfg.ClaudeProfiles) != 2 {
		t.Fatalf("want 2 profiles, got %d", len(cfg.ClaudeProfiles))
	}
	p0 := cfg.ClaudeProfiles[0]
	if p0.ID != "claude" || p0.ConfigDir != "/home/u/.claude" || p0.JSONPath != "/home/u/.claude.json" {
		t.Fatalf("p0 = %+v", p0)
	}
	p1 := cfg.ClaudeProfiles[1]
	if p1.ID != "claude-secondary" || p1.Label != "Claude (secondary)" || p1.ConfigDir != "/home/u/.claude-m" {
		t.Fatalf("p1 = %+v", p1)
	}
	// notify defaults still apply when [notify] is absent.
	if !cfg.NotifyEnabled {
		t.Fatal("expected notify default enabled")
	}
}

func TestParsePluginConfigTOML_NoProfilesByDefault(t *testing.T) {
	cfg := ParsePluginConfigTOML(DefaultPluginConfigTOML(DefaultPluginConfig))
	if len(cfg.ClaudeProfiles) != 0 {
		t.Fatalf("seed must not define active profiles, got %+v", cfg.ClaudeProfiles)
	}
}

func TestSeedPluginConfigIfMissing(t *testing.T) {
	dir := t.TempDir()
	if !SeedPluginConfigIfMissing(dir) {
		t.Fatal("first seed should write")
	}
	if SeedPluginConfigIfMissing(dir) {
		t.Fatal("second seed should not write")
	}
	text, err := os.ReadFile(filepath.Join(dir, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !contains(string(text), "[notify]") {
		t.Fatal("missing [notify]")
	}
	if !LoadPluginConfig(dir).NotifyEnabled {
		t.Fatal("expected enabled")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

func TestParsePluginConfigTOMLCodexProfiles(t *testing.T) {
	cfg := ParsePluginConfigTOML(`
[[codex.profiles]]
id = "almir"
label = "Codex almir"
codex_home = "/home/u/.codex-almir"

[[codex.profiles]]
id = "vvs"
codex_home = "/home/u/.codex-vvs"
`)
	if len(cfg.CodexProfiles) != 2 {
		t.Fatalf("want 2 codex profiles, got %d", len(cfg.CodexProfiles))
	}
	if cfg.CodexProfiles[0].ID != "almir" || cfg.CodexProfiles[0].Label != "Codex almir" ||
		cfg.CodexProfiles[0].CodexHome != "/home/u/.codex-almir" {
		t.Fatalf("first profile mis-decoded: %+v", cfg.CodexProfiles[0])
	}
	if cfg.CodexProfiles[1].Label != "" {
		t.Fatalf("absent label must stay empty for downstream id fallback, got %q", cfg.CodexProfiles[1].Label)
	}
}

func TestParsePluginConfigTOMLWithoutCodexSectionStaysEmpty(t *testing.T) {
	cfg := ParsePluginConfigTOML("[notify]\nenabled = true\n")
	if len(cfg.CodexProfiles) != 0 {
		t.Fatalf("no [[codex.profiles]] must yield none (implicit default downstream), got %+v", cfg.CodexProfiles)
	}
}

func TestDefaultPluginConfigTOMLDocumentsCodexProfiles(t *testing.T) {
	body := DefaultPluginConfigTOML(DefaultPluginConfig)
	for _, want := range []string{"[[codex.profiles]]", "codex_home"} {
		if !strings.Contains(body, want) {
			t.Fatalf("seed config must document %q", want)
		}
	}
	// The seed must stay commented out, or seeding would configure accounts.
	if strings.Contains(body, "\n[[codex.profiles]]") {
		t.Fatal("codex profile example must be commented out in the seed")
	}
}
