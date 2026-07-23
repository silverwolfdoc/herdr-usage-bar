package limits

import (
	"strings"
	"testing"
)

func TestFormatStatusBarShowsUsedPercentAndReset(t *testing.T) {
	now := int64(1_700_000_000_000)
	short := int64(now/1000 + 4*60*60 + 9*60)
	long := int64(now/1000 + 15*60*60 + 9*60)
	fiveHours := 300
	week := 10_080

	got := FormatStatusBar([]ProviderLimits{{
		ProviderID: "claude",
		Primary:    &LimitWindow{UsedPercentage: 5, ResetsAt: &short, WindowMinutes: &fiveHours},
		Secondary:  &LimitWindow{UsedPercentage: 80, ResetsAt: &long, WindowMinutes: &week},
	}}, now, StatusBarLayout{Columns: 120})

	for _, want := range []string{"✳", "5% used 4h 9m", "80% used 15h 9m"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
}

func TestFormatStatusBarFitsNarrowPane(t *testing.T) {
	reset := int64(1_700_000_600)
	window := 300
	got := FormatStatusBar([]ProviderLimits{{
		ProviderID: "claude",
		Primary:    &LimitWindow{UsedPercentage: 25, ResetsAt: &reset, WindowMinutes: &window},
	}, {
		ProviderID: "codex",
		Primary:    &LimitWindow{UsedPercentage: 50, ResetsAt: &reset, WindowMinutes: &window},
	}}, 1_700_000_000_000, StatusBarLayout{Columns: 24})

	if plainWidth(got) > 24 {
		t.Fatalf("status bar width %d exceeds pane: %q", plainWidth(got), got)
	}
	if strings.Contains(got, "\n") {
		t.Fatal("status bar must stay on one line")
	}
}
