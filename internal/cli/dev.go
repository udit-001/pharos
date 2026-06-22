package cli

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

// devFlags holds the flags for `pharos dev`.
var devFlags struct {
	port   int
	noOpen bool
}

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Hot-reload dev server (rebuilds Go + CSS on change)",
	Long: `Start a development server with live rebuild.

Two pipelines run in parallel:
  - Tailwind --watch regenerates web/app.css when web/input.css or any
    *.go source changes. The dev server serves CSS from disk, so styling
    edits are picked up on browser refresh without a Go rebuild.
  - fsnotify watches *.go and *.templ files. On change it rebuilds the
    binary to a temp path (so a failed build never kills the running
    server), then gracefully restarts the server via SIGINT.

Run from the project root. Ctrl-C stops both the tailwind and server
children cleanly.

Examples:
  pharos dev
  pharos dev --port 9090 --no-open`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDev()
	},
}

func init() {
	rootCmd.AddCommand(devCmd)
	devCmd.Flags().IntVar(&devFlags.port, "port", defaultPort, "HTTP server port")
	devCmd.Flags().BoolVar(&devFlags.noOpen, "no-open", false, "Don't auto-open browser")
}

// runDev orchestrates the tailwind watcher, the server subprocess, and the
// Go rebuild-on-change loop. The dev command itself never rebuilds; it
// manages a child `pharos start --foreground --dev-css` process.
func runDev() error {
	root := mustProjectRoot()
	tailwindBin := filepath.Join(root, ".local", "bin", "tailwindcss")
	devBin := filepath.Join(root, "tmp", "pharos-dev")
	tmpBin := devBin + ".next"

	if _, err := os.Stat(tailwindBin); err != nil {
		return fmt.Errorf("tailwind binary not found at %s — run `pharos dev` from the project root", tailwindBin)
	}
	if err := os.MkdirAll(filepath.Dir(devBin), 0o755); err != nil {
		return fmt.Errorf("create tmp dir: %w", err)
	}

	// 1. Tailwind child: regenerates web/app.css on input.css / *.go change.
	// --watch=always (not plain --watch): tailwind exits when stdin closes,
	// and this child gets /dev/null stdin. Without =always it builds once
	// then dies, so no CSS rebuilds ever happen.
	tw := execCommand(tailwindBin,
		"--input", "web/input.css",
		"--output", "web/app.css",
		"--content", "**/*.go",
		"--watch=always")
	tw.Dir = root
	tw.Stdout = os.Stdout
	tw.Stderr = os.Stderr
	if err := tw.Start(); err != nil {
		return fmt.Errorf("start tailwind: %w", err)
	}
	fmt.Println("[dev] tailwind watching web/input.css + *.go")

	// 2. Initial build + server start.
	if err := buildBinary(root, tmpBin); err != nil {
		_ = tw.Process.Signal(syscall.SIGTERM)
		return fmt.Errorf("initial build: %w", err)
	}
	if err := os.Rename(tmpBin, devBin); err != nil {
		return fmt.Errorf("swap dev binary: %w", err)
	}
	srv, err := startServer(root, devBin)
	if err != nil {
		_ = tw.Process.Signal(syscall.SIGTERM)
		return err
	}

	// 3. File watcher: *.go / *.templ under internal/ and cmd/, plus input.css.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("fsnotify: %w", err)
	}
	defer watcher.Close()
	for _, r := range []string{"internal", "cmd"} {
		addWatchDirs(watcher, filepath.Join(root, r))
	}
	// NOTE: web/input.css is intentionally NOT watched here. Tailwind --watch
	// rebuilds web/app.css on input.css change, and the dev server serves CSS
	// from disk — so styling edits never need a Go rebuild. Watching .css
	// here would just trigger redundant Go rebuilds on every CSS edit.

	// 4. Signal handling (Ctrl-C tears everything down).
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	// 5. Debounced rebuild loop.
	events := make(chan struct{}, 64)
	go func() {
		for {
			select {
			case ev, ok := <-watcher.Events:
				if !ok {
					return
				}
				if isRelevant(ev) {
					select {
					case events <- struct{}{}:
					default: // drop if backlogged; a rebuild is already pending
					}
				}
			case <-watcher.Errors:
				// ignore; fsnotify recovers internally on most errors
			}
		}
	}()

	debounce := time.NewTimer(0)
	if !debounce.Stop() {
		<-debounce.C
	}
	fmt.Printf("[dev] ready → http://127.0.0.1:%d  (Ctrl-C to stop)\n", devFlags.port)

	for {
		select {
		case <-events:
			debounce.Reset(250 * time.Millisecond)
		case <-debounce.C:
			fmt.Println("[dev] change detected, rebuilding…")
			// Build to a temp path first: a failed build leaves the
			// running server untouched.
			if err := buildBinary(root, tmpBin); err != nil {
				fmt.Printf("[dev] build failed: %v (server still running on old binary)\n", err)
				continue
			}
			stopServer(srv)
			if err := os.Rename(tmpBin, devBin); err != nil {
				fmt.Printf("[dev] swap binary failed: %v\n", err)
				continue
			}
			srv, err = startServer(root, devBin)
			if err != nil {
				fmt.Printf("[dev] restart failed: %v\n", err)
			} else {
				fmt.Printf("[dev] reloaded → http://127.0.0.1:%d\n", devFlags.port)
			}
		case <-sig:
			fmt.Println("\n[dev] shutting down…")
			stopServer(srv)
			_ = tw.Process.Signal(syscall.SIGTERM)
			_ = tw.Wait()
			return nil
		}
	}
}

// isRelevant returns true for write/create/rename events on Go/templ source
// files. CSS is excluded — tailwind owns the CSS pipeline and the dev server
// serves it from disk, so CSS edits must not trigger a Go rebuild.
func isRelevant(ev fsnotify.Event) bool {
	if ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) == 0 {
		return false
	}
	switch strings.ToLower(filepath.Ext(ev.Name)) {
	case ".go", ".templ":
		return true
	}
	return false
}

// addWatchDirs walks dir and adds every subdirectory to the watcher,
// skipping build/VC dirs. fsnotify is not recursive by default.
func addWatchDirs(w *fsnotify.Watcher, dir string) {
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		switch filepath.Base(path) {
		case "tmp", ".git", "node_modules":
			return filepath.SkipDir
		}
		return w.Add(path)
	})
}

// buildBinary compiles ./cmd/pharos to out.
func buildBinary(root, out string) error {
	cmd := exec.Command("go", "build", "-o", out, "./cmd/pharos")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// startServer runs the dev binary as `start --foreground --dev-css`.
func startServer(root, bin string) (*exec.Cmd, error) {
	args := []string{"start", "--foreground", "--dev-css",
		"--port", fmt.Sprintf("%d", devFlags.port)}
	if devFlags.noOpen {
		args = append(args, "--no-open")
	}
	c := execCommand(bin, args...)
	c.Dir = root
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Start(); err != nil {
		return nil, fmt.Errorf("start server: %w", err)
	}
	return c, nil
}

// stopServer sends SIGINT (graceful — server.go handles it) and waits up
// to 5s before SIGKILL.
func stopServer(srv *exec.Cmd) {
	if srv == nil || srv.Process == nil {
		return
	}
	done := make(chan struct{})
	go func() { _ = srv.Wait(); close(done) }()
	_ = srv.Process.Signal(syscall.SIGINT)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = srv.Process.Kill()
		<-done
	}
}

// mustProjectRoot returns the CWD; `pharos dev` is run from the project root.
func mustProjectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
