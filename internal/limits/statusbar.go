package limits

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/silverwolfdoc/herdr-usage-bar/internal/bar"
)

// StatusBarLayout controls the one-line usage pane renderer.
type StatusBarLayout struct {
	Columns int
	Color   bool
}

type statusBarMode uint8

const (
	statusBarFull statusBarMode = iota
	statusBarCompact
	statusBarTight
)

type statusBarWindow struct {
	used  int
	reset string
}

// FormatStatusBar renders provider quota windows as one compact line.
// Percentages are consumed percentage; reset text is a countdown.
func FormatStatusBar(providers []ProviderLimits, nowMs int64, layout StatusBarLayout) string {
	if nowMs == 0 {
		nowMs = time.Now().UnixMilli()
	}
	if layout.Columns < 1 {
		layout.Columns = 120
	}

	groups := make([][]statusBarWindow, 0, len(providers))
	icons := make([]string, 0, len(providers))
	for _, provider := range providers {
		windows := statusBarWindows(provider, nowMs)
		if len(windows) == 0 {
			continue
		}
		groups = append(groups, windows)
		icons = append(icons, statusBarIcon(provider.ProviderID))
	}
	if len(groups) == 0 {
		return ""
	}

	for _, mode := range []statusBarMode{statusBarFull, statusBarCompact, statusBarTight} {
		line := renderStatusBar(groups, icons, mode, layout.Color)
		if plainWidth(line) <= layout.Columns {
			return line
		}
	}

	// ponytail: narrow panes drop later providers instead of wrapping and
	// destroying the one-row surface; open limits pane retains full detail.
	return fitStatusBar(renderStatusBar(groups, icons, statusBarTight, layout.Color), layout.Columns)
}

func statusBarWindows(provider ProviderLimits, nowMs int64) []statusBarWindow {
	candidates := []*LimitWindow{provider.Primary, provider.Secondary, provider.Tertiary}
	windows := make([]statusBarWindow, 0, len(candidates))
	for _, window := range candidates {
		if window == nil {
			continue
		}
		if window.ResetsAt != nil && *window.ResetsAt > 0 && *window.ResetsAt <= nowMs/1000 {
			continue
		}
		used := int(math.Round(math.Max(0, math.Min(100, window.UsedPercentage))))
		reset := windowTag(window, "?")
		if window.ResetsAt != nil && *window.ResetsAt > 0 {
			reset = formatResetIn(*window.ResetsAt*1000 - nowMs)
		}
		windows = append(windows, statusBarWindow{used: used, reset: reset})
	}
	return windows
}

func statusBarIcon(providerID string) string {
	id := strings.ToLower(providerID)
	switch {
	case strings.HasPrefix(id, "claude"):
		return "✳"
	case strings.HasPrefix(id, "codex"):
		return "◉"
	case strings.HasPrefix(id, "opencode"):
		return "◆"
	case strings.HasPrefix(id, "grok"):
		return "𝕏"
	default:
		return "•"
	}
}

func renderStatusBar(groups [][]statusBarWindow, icons []string, mode statusBarMode, color bool) string {
	parts := make([]string, 0, len(groups))
	for i, windows := range groups {
		parts = append(parts, renderStatusBarGroup(icons[i], windows, mode, color))
	}
	return strings.Join(parts, "  ·  ")
}

func renderStatusBarGroup(icon string, windows []statusBarWindow, mode statusBarMode, color bool) string {
	windowParts := make([]string, 0, len(windows))
	for _, window := range windows {
		windowParts = append(windowParts, formatStatusBarWindow(window, mode))
	}

	if mode == statusBarTight {
		return icon + " " + strings.Join(windowParts, " · ")
	}

	trackWidth := 8
	if mode == statusBarCompact {
		trackWidth = 5
	}
	remaining := float64(100 - windows[0].used)
	track := bar.RenderBar(remaining, trackWidth)
	if color {
		track = bar.Colorize(track, bar.ToneForRemaining(remaining), true)
	}
	return icon + " " + track + " " + strings.Join(windowParts, " · ")
}

func formatStatusBarWindow(window statusBarWindow, mode statusBarMode) string {
	switch mode {
	case statusBarFull:
		return fmt.Sprintf("%d%% used %s", window.used, window.reset)
	case statusBarCompact:
		return fmt.Sprintf("%d%% %s", window.used, window.reset)
	default:
		return fmt.Sprintf("%d%%/%s", window.used, window.reset)
	}
}

func fitStatusBar(line string, columns int) string {
	if columns < 1 {
		return ""
	}
	if plainWidth(line) <= columns {
		return line
	}
	plain := ansiRe.ReplaceAllString(line, "")
	runes := []rune(plain)
	if len(runes) <= columns {
		return plain
	}
	if columns == 1 {
		return "…"
	}
	return string(runes[:columns-1]) + "…"
}
