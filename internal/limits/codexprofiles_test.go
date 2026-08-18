package limits

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/silverwolfdoc/herdr-usage-bar/internal/codex"
)

func writeCodexAuth(t *testing.T, dir, email string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]string{"email": email})
	idToken := "h." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
	body, _ := json.Marshal(map[string]any{"tokens": map[string]string{"id_token": idToken}})
	path := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCodexAccountEmailFromAuthPath(t *testing.T) {
	dir := t.TempDir()
	path := writeCodexAuth(t, dir, "almir@example.com")
	got, ok := CodexAccountEmailFromAuthPath(path)
	if !ok || got != "almir@example.com" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
}

func TestCodexAccountEmailFromAuthPathDegradesSafely(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"missing file": filepath.Join(dir, "nope.json"),
		"not json":     writeRaw(t, filepath.Join(dir, "bad.json"), "not json"),
		"no tokens":    writeRaw(t, filepath.Join(dir, "empty.json"), `{}`),
		"bad jwt":      writeRaw(t, filepath.Join(dir, "jwt.json"), `{"tokens":{"id_token":"only-one-part"}}`),
		"no email":     writeRaw(t, filepath.Join(dir, "noemail.json"), `{"tokens":{"id_token":"h.e30.s"}}`),
	}
	for name, path := range cases {
		if got, ok := CodexAccountEmailFromAuthPath(path); ok {
			t.Fatalf("%s: expected no email, got %q", name, got)
		}
	}
}

func writeRaw(t *testing.T, path, body string) string {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestApplyCodexProfileGroupingSingleProfileIsUntouched(t *testing.T) {
	p := codex.CodexProfile{ID: "codex", Label: "Codex"}
	pl := ProviderLimits{ProviderID: "codex", Label: "Codex"}
	got := applyCodexProfileGrouping(pl, p, false)
	if got.GroupLabel != "" || got.AccountLabel != "" {
		t.Fatalf("single profile must not be grouped: %+v", got)
	}
}

func TestApplyCodexProfileGroupingUsesAccountEmail(t *testing.T) {
	dir := t.TempDir()
	writeCodexAuth(t, dir, "vvs@example.com")
	p := codex.CodexProfile{ID: "vvs", Label: "Codex vvs", AuthPath: filepath.Join(dir, "auth.json")}
	got := applyCodexProfileGrouping(ProviderLimits{ProviderID: "vvs", Label: "Codex vvs"}, p, true)
	if got.GroupLabel != "Codex" {
		t.Fatalf("GroupLabel = %q, want Codex", got.GroupLabel)
	}
	if got.AccountLabel != "vvs@example.com" {
		t.Fatalf("AccountLabel = %q, want the logged-in email", got.AccountLabel)
	}
}

func TestApplyCodexProfileGroupingFallsBackToLabelWhenAuthUnreadable(t *testing.T) {
	p := codex.CodexProfile{ID: "vvs", Label: "Codex vvs", AuthPath: "/nonexistent/auth.json"}
	got := applyCodexProfileGrouping(ProviderLimits{ProviderID: "vvs", Label: "Codex vvs"}, p, true)
	if got.AccountLabel != "Codex vvs" {
		t.Fatalf("AccountLabel = %q, want the profile label fallback", got.AccountLabel)
	}
}
