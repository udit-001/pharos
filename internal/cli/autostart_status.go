package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/autostart"
)

var autostartStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show whether Pharos starts at login",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		m, _, err := autostartManagerFor(cmd)
		if err != nil {
			return err
		}
		r := m.Status()
		if jsonEnabled(cmd) {
			printJSON(autostartJSON{
				Enabled:   r.Enabled,
				Status:    string(r.Status),
				EntryPath: r.EntryPath,
				Port:      r.Port,
			})
			return nil
		}
		fmt.Println()
		fmt.Printf("  Autostart: %s\n", r.Status)
		fmt.Printf("    Entry:   %s\n", r.EntryPath)
		if r.Enabled && r.Port != 0 {
			fmt.Printf("    Port:    %d\n", r.Port)
		}
		if r.Status == autostart.StatusStalePort {
			fmt.Println("  Fix:      pharos autostart enable")
		}
		fmt.Println()
		return nil
	},
}
