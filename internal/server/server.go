package server

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/udit-001/pharos/internal/db"
)

// Config holds the server's startup configuration.
type Config struct {
	Port   int
	DB     *db.Store
	NoOpen bool
	Silent bool
	DevCSS bool // serve CSS from disk (no embed, no-cache) for `pharos dev`
}

// Start boots the dashboard server: builds the mux via NewMux, finds a free
// port, optionally opens the browser, and serves until SIGINT/SIGTERM.
//
// The mux is a separate seam (NewMux) so tests can drive routes through
// httptest without booting a real listener.
func Start(cfg Config) error {
	mux := NewMux(cfg.DB, cfg.DevCSS)

	// Try the configured port, then port+1, port+2, … up to 100 attempts.
	var listener net.Listener
	port := cfg.Port
	for i := 0; i < 100; i++ {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		var err error
		listener, err = net.Listen("tcp", addr)
		if err == nil {
			break
		}
		port++
	}
	if listener == nil {
		return fmt.Errorf("no free port found starting from %d", cfg.Port)
	}
	cfg.Port = port // store the actual port for messages below

	srv := newServer(mux)

	if !cfg.NoOpen && !cfg.Silent {
		url := fmt.Sprintf("http://127.0.0.1:%d", port)
		if err := openBrowser(url); err != nil {
			log.Printf("  Open %s in your browser", url)
		}
	}
	if cfg.Silent {
		log.Printf("Pharos listening on http://127.0.0.1:%d", port)
	} else {
		fmt.Printf("  Pharos Dashboard: http://127.0.0.1:%d\n", port)
		fmt.Println()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-quit; srv.Close() }()
	// http.ErrServerClosed is the expected return from a graceful SIGINT/SIGTERM
	// shutdown (srv.Close above) — treat it as a clean exit, not an error,
	// so `pharos start`/`pharos dev` don't print a spurious error on stop.
	if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// newServer builds the *http.Server with transport-layer timeouts. Extracted
// from Start so the timeout wiring is testable without booting a listener.
// These bound how long a stuck client can hold a request goroutine (and the
// DB query behind it) — the transport backstop for the per-query timeouts
// tracked in LEARN-101 (LEARN-104). Values suit a localhost dashboard; the
// slowest legitimate response is a full-workspace Search, which stays well
// under WriteTimeout on personal-sized DBs.
func newServer(mux *http.ServeMux) *http.Server {
	return &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

// openBrowser launches the OS default browser at the given URL.
func openBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	return exec.Command(cmd, args...).Start()
}
