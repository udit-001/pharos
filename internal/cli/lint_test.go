package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

// runCmdBoth drives a cobra subcommand with an injected store and returns
// (stdout, stderr, err). Used by the lint tests: warnings print to stderr
// while the command still succeeds.
func runCmdBoth(t *testing.T, args []string, store *db.Store) (string, string, error) {
	t.Helper()
	root := newRootForTest()
	ctx := context.WithValue(context.Background(), ctxStore{}, store)
	root.SetArgs(args)
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(context.WithValue(cmd.Context(), ctxStore{}, store))
		return nil
	}
	root.PersistentPostRunE = nil

	var outBuf, errBuf bytes.Buffer
	origOut, origErr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr
	defer func() { os.Stdout, os.Stderr = origOut, origErr }()

	err := root.ExecuteContext(ctx)
	wOut.Close()
	wErr.Close()
	_, _ = outBuf.ReadFrom(rOut)
	_, _ = errBuf.ReadFrom(rErr)
	return outBuf.String(), errBuf.String(), err
}

// writeBody writes a temp body file and returns its path.
func writeBody(t *testing.T, html string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "body.html")
	if err := os.WriteFile(p, []byte(html), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLintQuizWarnings(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ws, err := store.AddWorkspace(db.Workspace{Name: "lint-ws", Path: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	// Quiz with no .fb and a data-answer matching no option — two warnings,
	// but the lesson still gets created (warnings are advisory).
	bad := writeBody(t, `<html><head></head><body><div class="q" data-answer="B"><p>Q</p><div class="options"><button>A</button></div></div></body></html>`)
	_, stderr, err := runCmdBoth(t, []string{"lesson", "create", "Bad Quiz", "-w", ws.Name, "--body-file", bad}, store)
	if err != nil {
		t.Fatalf("create should succeed despite warnings: %v", err)
	}
	if !strings.Contains(stderr, "data-answer") {
		t.Errorf("warning should mention unmatched data-answer; stderr: %s", stderr)
	}
	if !strings.Contains(stderr, ".fb") {
		t.Errorf("warning should mention missing .fb; stderr: %s", stderr)
	}

	// --force silences the lint.
	force := writeBody(t, `<html><head></head><body><div class="q" data-answer="B"><p>Q</p><div class="options"><button>A</button></div></div></body></html>`)
	_, stderr, err = runCmdBoth(t, []string{"lesson", "create", "Forced Quiz", "-w", ws.Name, "--body-file", force, "--force"}, store)
	if err != nil {
		t.Fatalf("create --force should succeed: %v", err)
	}
	if strings.Contains(stderr, "warning") {
		t.Errorf("--force should silence lint warnings; stderr: %s", stderr)
	}

	// Clean quiz — no warnings.
	good := writeBody(t, `<html><head></head><body><div class="q" data-answer="A"><p>Q</p><div class="options"><button>A</button></div><div class="fb"></div></div></body></html>`)
	_, stderr, err = runCmdBoth(t, []string{"lesson", "create", "Good Quiz", "-w", ws.Name, "--body-file", good}, store)
	if err != nil {
		t.Fatalf("clean create failed: %v", err)
	}
	if strings.Contains(stderr, "warning") {
		t.Errorf("clean lesson should print no warnings; stderr: %s", stderr)
	}
}

func TestLintRuleCoverage(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ws, err := store.AddWorkspace(db.Workspace{Name: "lint-ws2", Path: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name  string
		html  string
		want  string // substring the stderr must contain
		clean bool   // true when this fixture must NOT warn
	}{
		{name: "vega container without spec", html: `<html><body><div class="chart" data-vega="c1"></div></body></html>`, want: `chart "c1"`},
		{name: "spec without container", html: `<html><body><script type="application/json" id="orphan">{}</script></body></html>`, want: `chart spec "orphan"`},
		{name: "parent-relative asset path", html: `<html><body><img src="../assets/x.png"></body></html>`, want: "../"},
		{name: "dashboard route without _top", html: `<html><body><a href="/workspace/x/lesson/y">next</a></body></html>`, want: `target="_top"`},
		{name: "external link without noopener", html: `<html><body><a href="https://example.com">x</a></body></html>`, want: "noopener"},
		{name: "CDN script", html: `<html><head><script src="https://cdn.jsdelivr.net/npm/x.js"></script></head><body></body></html>`, want: "CDN"},
		{name: "glossary span without data-term", html: `<html><body><span class="glossary-term">join</span></body></html>`, want: "data-term"},
		{name: "well-formed vega pair", clean: true, html: `<html><body><div class="chart" data-vega="ok"></div><script type="application/json" id="ok">{}</script></body></html>`},
		{name: "top-targeted dashboard link + noopener external", clean: true, html: `<html><body><a href="/workspace/x/lesson/y" target="_top">n</a><a href="https://example.com" target="_blank" rel="noopener noreferrer">e</a></body></html>`},
		{name: "glossary with term", clean: true, html: `<html><body><span class="glossary-term" data-term="join">join</span></body></html>`},
	}

	for i, tc := range cases {
		body := writeBody(t, tc.html)
		_, stderr, err := runCmdBoth(t, []string{"lesson", "create", fmt.Sprintf("Lint Case %d", i), "-w", ws.Name, "--body-file", body}, store)
		if err != nil {
			t.Fatalf("%s: create failed: %v", tc.name, err)
		}
		if tc.clean {
			if strings.Contains(stderr, "warning") {
				t.Errorf("%s: expected clean, got warnings: %s", tc.name, stderr)
			}
			continue
		}
		if !strings.Contains(stderr, tc.want) {
			t.Errorf("%s: stderr missing %q; got: %s", tc.name, tc.want, stderr)
		}
	}
}

func TestLintMissingAssetReference(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	ws, err := store.AddWorkspace(db.Workspace{Name: "lint-ws3", Path: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	// Referenced asset exists on disk — no warning.
	os.MkdirAll(filepath.Join(ws.Path, "assets"), 0755)
	os.WriteFile(filepath.Join(ws.Path, "assets", "pic.png"), []byte("x"), 0644)

	bad := writeBody(t, `<html><body><img src="assets/missing.png"></body></html>`)
	_, stderr, err := runCmdBoth(t, []string{"lesson", "create", "Missing Asset", "-w", ws.Name, "--body-file", bad}, store)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr, "missing.png") {
		t.Errorf("missing asset should warn; stderr: %s", stderr)
	}

	good := writeBody(t, `<html><body><img src="assets/pic.png"></body></html>`)
	_, stderr, err = runCmdBoth(t, []string{"lesson", "create", "Existing Asset", "-w", ws.Name, "--body-file", good}, store)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stderr, "warning") {
		t.Errorf("existing asset should not warn; stderr: %s", stderr)
	}
}
