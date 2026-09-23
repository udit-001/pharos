package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/autostart"
	"github.com/udit-001/pharos/internal/config"
)

// pharos setup — the single restore command for the desktop-app story
// (LEARN-166 #281, LEARN-221). Fixed order: server (hard gate, exit 1) →
// autostart (soft) → PWA/browser (soft). One report struct, two renderers,
// no drift. No new flags; global --json inherited. Port is an identity:
// never auto-incremented; a busy port is a hard gate.

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Restore the desktop setup: server, autostart, PWA install",
	Long: `Restore the full desktop-app setup with one command:

  1. Ensure the Pharos server is running on the configured port
     (restarting it if it is on a different port). Hard gate: exit 1 if the
     configured port cannot be served.
  2. Enable Pharos at login (always rewrites the per-user startup entry
     with the current port). Soft step.
  3. Open the /setup PWA install page in a Chromium browser (Edge on
     Windows, Chrome/Edge/Brave/Chromium/helium on macOS/Linux), falling
     back to the default browser, then to printing the URL. Machines with no
     Chromium get a desktop launcher instead of the page. Skip everything
     when the PWA is already installed for this origin. Soft step; the URL
     is always printed.

Examples:
  pharos setup`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		port := resolvePort(cmd)
		rep, ok := runSetup(port, setupRunner{})
		if jsonEnabled(cmd) {
			printJSON(rep)
		} else {
			fmt.Print(renderSetupHuman(rep))
		}
		if !ok {
			// Server = hard gate: report already printed, exit non-zero
			// (repo precedent: os.Exit in nav.go).
			os.Exit(1)
		}
		return nil
	},
}

// ── report (one struct, two renderers) ────────────────────────────────────

type setupServer struct {
	Running  bool   `json:"running"`
	Port     int    `json:"port"`
	URL      string `json:"url"`
	Action   string `json:"-"` // console-only: started | running | restarted | failed
	FailText string `json:"-"` // console-only: hard-gate reason
}

type setupAutostart struct {
	Status    string `json:"status"` // enabled | disabled | failed (LEARN-165 vocabulary)
	EntryPath string `json:"entry_path"`
	Port      int    `json:"port"`
	PrevPort  int    `json:"-"` // console-only: "rewritten for N (was M)"
	Err       string `json:"-"` // console-only warning detail
}

type setupPWA struct {
	Installed bool   `json:"installed"`
	Origin    string `json:"origin"`
}

type setupBrowser struct {
	Opened          bool   `json:"opened"`
	Method          string `json:"method"` // chromium | default | none
	ShortcutCreated bool   `json:"shortcut_created"`
	URL             string `json:"url"`
	Name            string `json:"-"` // console-only browser name
}

type setupReport struct {
	Server    setupServer     `json:"server"`
	Autostart *setupAutostart `json:"autostart,omitempty"` // nil on a server gate
	PWA       *setupPWA       `json:"pwa,omitempty"`
	Browser   *setupBrowser   `json:"browser,omitempty"`
}

// ── orchestration ─────────────────────────────────────────────────────────

// setupRunner is the side-effect seam for runSetup: the small set of
// process/browser operations a unit test cannot perform honestly
// (spawn/kill the daemon, open the browser). Everything else — pidfile,
// pwa.json, autostart entry, config — is real, pointed at temp dirs via
// XDG_CONFIG_HOME (LEARN-166 #281 "real temp ConfigDirs, stub the
// killer/opener").
type setupRunner struct {
	kill     func() error
	start    func(port int) (*exec.Cmd, error)
	serverUp func(port int) bool
	openPWA  func(port int) browserResult
}

func (r *setupRunner) fillDefaults() {
	if r.kill == nil {
		// Outcome-based: the port-change branch has already verified a
		// healthy pidfile, so a "nothing to stop" verdict here means the
		// pidfile vanished mid-run — surface it as a gate rather than
		// pretending all is well.
		r.kill = func() error {
			switch stopServerByPidfile() {
			case stopStopped, stopAlreadyStopped, stopStalePID, stopForeignPID:
				return nil
			default:
				return fmt.Errorf("no server found to stop")
			}
		}
	}
	if r.start == nil {
		r.start = startDaemon
	}
	if r.serverUp == nil {
		r.serverUp = waitForServer
	}
	if r.openPWA == nil {
		r.openPWA = openSetupPageDefault
	}
}

// runSetup executes the fixed flow. ok=false means the server gate failed —
// the caller exits 1 with the (server-only) report already rendered.
func runSetup(port int, r setupRunner) (setupReport, bool) {
	r.fillDefaults()
	rep := setupReport{}
	rootURL := dashboardURLFor(port)
	origin := strings.TrimSuffix(rootURL, "/")

	// 1. Server — hard gate.
	info, perr := readPidFile()
	switch {
	case perr == nil && info.Port == port && processAlive(info.PID) && isServerRunning(port):
		rep.Server = setupServer{Running: true, Port: port, URL: rootURL, Action: "running"}
	case perr == nil && info.Port != port && processAlive(info.PID) && isServerRunning(info.Port):
		// Serving on a different port → kill + restart, the same path as
		// `pharos stop` (stopProcess + cleanupPidFile), then startDaemon +
		// pidfile poll (LEARN-166 Q3). The killed server holds info.Port,
		// never the target port, so there is no listener-release race with
		// the new daemon's bind.
		if err := r.kill(); err != nil {
			rep.Server = setupServer{Running: false, Port: port, URL: rootURL, Action: "failed", FailText: fmt.Sprintf("could not stop the server on port %d", info.Port)}
			return rep, false
		}
		if _, err := r.start(port); err != nil {
			rep.Server = setupServer{Running: false, Port: port, URL: rootURL, Action: "failed", FailText: err.Error()}
			return rep, false
		}
		if !r.serverUp(port) {
			rep.Server = setupServer{Running: false, Port: port, URL: rootURL, Action: "failed", FailText: gateReason(port)}
			return rep, false
		}
		rep.Server = setupServer{Running: true, Port: port, URL: rootURL, Action: "restarted"}
	default:
		if _, err := r.start(port); err != nil {
			rep.Server = setupServer{Running: false, Port: port, URL: rootURL, Action: "failed", FailText: err.Error()}
			return rep, false
		}
		if !r.serverUp(port) {
			rep.Server = setupServer{Running: false, Port: port, URL: rootURL, Action: "failed", FailText: gateReason(port)}
			return rep, false
		}
		rep.Server = setupServer{Running: true, Port: port, URL: rootURL, Action: "started"}
	}

	// 2. Autostart — soft (LEARN-165 enable semantics: always rewrite with
	// the current port; the report shows a self-healing rewrite as
	// "rewritten for N (was M)").
	am, err := autostart.New(autostart.Options{Port: port})
	if err != nil {
		rep.Autostart = &setupAutostart{Status: "failed", Port: port, Err: err.Error()}
	} else if prev, err := am.Enable(); err != nil {
		rep.Autostart = &setupAutostart{Status: "failed", EntryPath: am.EntryPath(), Port: port, Err: err.Error()}
	} else {
		rep.Autostart = &setupAutostart{Status: "enabled", EntryPath: am.EntryPath(), Port: port, PrevPort: prev}
	}

	// 3. PWA / browser — soft (LEARN-170 ladder). pwa.json installed at the
	// current origin → skip the browser entirely.
	if rec, _ := config.ReadPWAFile(); rec != nil && rec.Installed && rec.Origin == origin {
		rep.PWA = &setupPWA{Installed: true, Origin: rec.Origin}
		rep.Browser = &setupBrowser{Method: "none", URL: rootURL}
	} else {
		rep.PWA = &setupPWA{Origin: origin}
		b := r.openPWA(port)
		rep.Browser = &setupBrowser{Opened: b.opened, Method: b.method, ShortcutCreated: b.shortcutCreated, URL: b.url, Name: b.name}
	}
	return rep, true
}

// waitForServer polls the pid file + health check until the daemon serves
// the port (~2s budget, LEARN-166 Q3 "poll pidfile ~2s").
func waitForServer(port int) bool {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if info, err := readPidFile(); err == nil && info.Port == port && processAlive(info.PID) && isServerRunning(port) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// gateReason explains why the configured port isn't being served: a foreign
// process holding it, or the daemon never coming up.
func gateReason(port int) string {
	if isServerRunning(port) {
		return fmt.Sprintf("port %d is in use by another process", port)
	}
	return fmt.Sprintf("the daemon did not come up on port %d", port)
}

// ── console renderer (LEARN-166 #281 exact shapes) ────────────────────────

func renderSetupHuman(rep setupReport) string {
	var b strings.Builder
	label := func(name, val string) {
		fmt.Fprintf(&b, "%-10s %s\n", name+":", val)
	}

	if rep.Server.Action == "failed" {
		// Hard gate: server line + one fix line (LEARN-166 #281, LEARN-221).
		label("Server", "failed -- "+rep.Server.FailText)
		label("Fix", "free port "+strconv.Itoa(rep.Server.Port)+", then run pharos setup again")
		return b.String()
	}

	serverMood := map[string]string{
		"started":   fmt.Sprintf("started on %d", rep.Server.Port),
		"running":   fmt.Sprintf("running on %d", rep.Server.Port),
		"restarted": fmt.Sprintf("starting on %d ... OK", rep.Server.Port),
	}[rep.Server.Action]
	label("Server", serverMood)

	// Autostart mood: rewrite (port change) > enabled > failed.
	switch {
	case rep.Autostart.Status == "failed":
		label("Autostart", "failed -- "+rep.Autostart.Err)
	case rep.Autostart.PrevPort > 0 && rep.Autostart.PrevPort != rep.Autostart.Port:
		label("Autostart", fmt.Sprintf("rewritten for %d (was %d)", rep.Autostart.Port, rep.Autostart.PrevPort))
	default:
		label("Autostart", rep.Autostart.Status)
	}

	// PWA/browser mood.
	var pwaMood string
	switch {
	case rep.PWA.Installed:
		pwaMood = "already installed -- dashboard: " + rep.Browser.URL
	case rep.Browser.Opened && rep.Browser.Method == "chromium":
		if rep.Server.Action == "restarted" {
			pwaMood = fmt.Sprintf("re-install from %s (opening %s)", rep.Browser.URL, rep.Browser.Name)
		} else {
			pwaMood = fmt.Sprintf("opening %s -> %s", rep.Browser.Name, rep.Browser.URL)
		}
	case rep.Browser.Opened && rep.Browser.Method == "default":
		pwaMood = "opening default browser -> " + rep.Browser.URL
	case rep.Browser.ShortcutCreated:
		pwaMood = "no Chromium found -- created Pharos.url on your Desktop"
	default:
		pwaMood = "no Chromium found -- open " + rep.Browser.URL + " manually"
	}
	label("PWA", pwaMood)
	return b.String()
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
