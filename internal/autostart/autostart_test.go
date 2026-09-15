package autostart

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── daemon args ──────────────────────────────────────────────────────────

func TestArgsPinPort(t *testing.T) {
	got := args(9091)
	want := []string{"start", "--background", "--no-open", "--daemon", "--port", "9091"}
	if len(got) != len(want) {
		t.Fatalf("args(%d) = %v, want %v", 9091, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("args(%d) = %v, want %v", 9091, got, want)
		}
	}
}

func TestPortFromArgs(t *testing.T) {
	cases := []struct {
		args []string
		want int
	}{
		{args(9090), 9090},
		{[]string{"start", "--no-open", "--daemon"}, 0},
		{[]string{"start", "--port", "abc"}, 0},
		{[]string{"start", "--port"}, 0},
	}
	for _, c := range cases {
		if got := portFromArgs(c.args); got != c.want {
			t.Errorf("portFromArgs(%v) = %d, want %d", c.args, got, c.want)
		}
	}
}

// ── content generators (pure, platform-neutral) ───────────────────────────

func TestPlistContent(t *testing.T) {
	xml := string(plistContent("/usr/local/bin/pharos", args(9090)))
	for _, want := range []string{
		"<key>Label</key>",
		"<string>com.udit001.pharos</string>",
		"/usr/local/bin/pharos",
		"start",
		"--background",
		"--no-open",
		"--daemon",
		"--port",
		"9090",
		"<key>RunAtLoad</key>",
		"<true/>",
		"<key>KeepAlive</key>",
		"<false/>",
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("plist missing %q:\n%s", want, xml)
		}
	}
}

func TestDesktopContent(t *testing.T) {
	text := string(desktopContent("/usr/local/bin/pharos", args(9091)))
	for _, want := range []string{
		"[Desktop Entry]",
		"Type=Application",
		"Name=Pharos",
		`Exec="/usr/local/bin/pharos" start --background --no-open --daemon --port 9091`,
		"Terminal=false",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("desktop file missing %q:\n%s", want, text)
		}
	}
}

func TestLnkScript(t *testing.T) {
	script := lnkScript(`C:\Program Files\Pharos\pharos.exe`, `C:\Users\me\Startup\pharos.lnk`, 9092)
	for _, want := range []string{
		"CreateShortcut",
		"'C:\\Users\\me\\Startup\\pharos.lnk'",
		"TargetPath = 'C:\\Program Files\\Pharos\\pharos.exe'",
		"Arguments = 'start --background --no-open --daemon --port 9092'",
		"Save()",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("lnk script missing %q:\n%s", want, script)
		}
	}
	// The console flash is accepted by design — a code comment must exist in
	// the Windows generator so nobody "fixes" it later (LEARN-165 #280).
	src, err := os.ReadFile(filepath.Join("windows.go"))
	if err == nil && !strings.Contains(string(src), "flash") {
		t.Error("windows.go must carry a comment explaining the accepted console flash (LEARN-165 #280)")
	}
}

func TestPsQuote(t *testing.T) {
	if got := psQuote("O'Brien"); got != "'O''Brien'" {
		t.Errorf("psQuote(O'Brien) = %s, want 'O''Brien'", got)
	}
}

// ── parsers (read-back of what we write) ──────────────────────────────────

func TestParsePlistArgs(t *testing.T) {
	got := parsePlistArgs(string(plistContent("/usr/bin/pharos", args(9090))))
	want := []string{"/usr/bin/pharos", "start", "--background", "--no-open", "--daemon", "--port", "9090"}
	if len(got) != len(want) {
		t.Fatalf("parsePlistArgs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parsePlistArgs = %v, want %v", got, want)
		}
	}
	if portFromArgs(got) != 9090 {
		t.Fatalf("roundtrip port = %d, want 9090", portFromArgs(got))
	}
}

func TestParseDesktopArgs(t *testing.T) {
	got := parseDesktopArgs(string(desktopContent("/usr/bin/pharos", args(9090))))
	if len(got) < 2 || got[1] != "start" {
		t.Fatalf("parseDesktopArgs = %v", got)
	}
	if portFromArgs(got) != 9090 {
		t.Fatalf("roundtrip port = %d, want 9090", portFromArgs(got))
	}
}

func TestSplitQuotedHandlesEscapedQuotes(t *testing.T) {
	// strconv.Quote escapes an embedded double quote as \" — the parser
	// must not treat it as a quote delimiter (pathological but round-trippable).
	got := splitQuoted(`"/usr/bin/phar\"os" start --daemon`)
	if len(got) != 3 || got[0] != `/usr/bin/phar"os` {
		t.Fatalf("splitQuoted = %q", got)
	}
}

// ── Manager state transitions (real temp dirs, no mocks) ──────────────────

func newManager(t *testing.T, port int) (*Manager, string) {
	t.Helper()
	dir := t.TempDir()
	entry := filepath.Join(dir, "pharos.desktop")
	m, err := New(Options{Exe: "/usr/bin/pharos", Port: port, EntryPath: entry})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return m, entry
}

func TestStatusDisabledWhenNoEntry(t *testing.T) {
	m, _ := newManager(t, 9090)
	r := m.Status()
	if r.Enabled || r.Status != StatusDisabled || r.Port != 0 {
		t.Fatalf("Status = %+v, want disabled", r)
	}
}

func TestEnableThenStatusEnabled(t *testing.T) {
	m, entry := newManager(t, 9090)
	prev, err := m.Enable()
	if err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if prev != 0 {
		t.Fatalf("Enable prev = %d, want 0", prev)
	}
	if _, err := os.Stat(entry); err != nil {
		t.Fatalf("entry not written: %v", err)
	}
	r := m.Status()
	if !r.Enabled || r.Status != StatusEnabled || r.Port != 9090 {
		t.Fatalf("Status = %+v, want enabled@9090", r)
	}
}

func TestEnableReturnsPrevPort(t *testing.T) {
	m, _ := newManager(t, 9091)
	if _, err := m.Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	m2, _ := newManager(t, 9091)
	m2.entryPath = m.entryPath
	prev, err := m2.Enable()
	if err != nil {
		t.Fatalf("Enable (rewrite): %v", err)
	}
	if prev != 9091 {
		t.Fatalf("Enable prev = %d, want 9091 (self-healing rewrite)", prev)
	}
}

func TestStatusStalePort(t *testing.T) {
	m, _ := newManager(t, 9090)
	if _, err := m.Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	stale, _ := New(Options{Exe: "/usr/bin/pharos", Port: 9091, EntryPath: m.entryPath})
	r := stale.Status()
	if !r.Enabled || r.Status != StatusStalePort || r.Port != 9090 {
		t.Fatalf("Status = %+v, want enabled (stale port) with embedded 9090", r)
	}
}

func TestDisableRemovesEntry(t *testing.T) {
	m, entry := newManager(t, 9090)
	if _, err := m.Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if err := m.Disable(); err != nil {
		t.Fatalf("Disable: %v", err)
	}
	if _, err := os.Stat(entry); !os.IsNotExist(err) {
		t.Fatalf("entry still exists after disable")
	}
	// Disable when already disabled is a no-op success.
	if err := m.Disable(); err != nil {
		t.Fatalf("Disable (no entry): %v", err)
	}
}

func TestNewResolvesExeWhenEmpty(t *testing.T) {
	m, err := New(Options{Port: 9090, EntryPath: filepath.Join(t.TempDir(), "e")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if m.exe == "" {
		t.Fatal("exe not resolved")
	}
}
