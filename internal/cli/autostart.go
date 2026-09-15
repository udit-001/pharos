package cli

import (
	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/autostart"
)

// pharos autostart manages the per-user startup entry (LEARN-222).
//
// The entry runs the identical daemon command line a user runs manually —
// `pharos start --background --no-open --daemon` — pinned to the configured
// port at enable time, per-platform (macOS LaunchAgents plist, Linux XDG
// autostart desktop file, Windows Startup shortcut).
var autostartCmd = &cobra.Command{
	Use:   "autostart",
	Short: "Manage the per-user startup entry (start Pharos at login)",
	Long: `Manage the per-user startup entry that starts the Pharos dashboard at login.

Writes a per-user entry (no admin, no service):
  macOS  ~/Library/LaunchAgents/com.udit001.pharos.plist
  Linux  ~/.config/autostart/pharos.desktop
  Windows  %APPDATA%\...\Startup\pharos.lnk

The entry pins the configured port at enable time; 'pharos autostart status'
reports "enabled (stale port)" when the port has since changed — run
'pharos autostart enable' (or 'pharos setup', the guided installer) to
rewrite it.

Examples:
  pharos autostart enable
  pharos autostart disable
  pharos autostart status`,
	Args: cobra.NoArgs,
	RunE: runShowHelp,
}

// autostartJSON is the --json status shape shared by enable/disable/status
// and echoed by `pharos setup` (LEARN-166 #281: enabled|disabled|failed,
// stale inside the status string). A struct — not a map — so key order is
// stable across runs.
type autostartJSON struct {
	Enabled   bool   `json:"enabled"`
	Status    string `json:"status"`
	EntryPath string `json:"entry_path"`
	Port      int    `json:"port"`
}

// autostartManagerFor builds the per-platform manager and resolves the
// configured port once (flag > config > default — same resolution as
// `pharos start`). Resolving here instead of per-command avoids a second
// config.Load in the leaves.
func autostartManagerFor(cmd *cobra.Command) (*autostart.Manager, int, error) {
	port := resolvePort(cmd)
	m, err := autostart.New(autostart.Options{Port: port})
	if err != nil {
		return nil, 0, err
	}
	return m, port, nil
}

func init() {
	rootCmd.AddCommand(autostartCmd)
	autostartCmd.AddCommand(autostartEnableCmd)
	autostartCmd.AddCommand(autostartDisableCmd)
	autostartCmd.AddCommand(autostartStatusCmd)
}
