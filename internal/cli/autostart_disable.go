package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var autostartDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable Pharos at login (remove the startup entry)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		m, _, err := autostartManagerFor(cmd)
		if err != nil {
			return err
		}
		if err := m.Disable(); err != nil {
			return fmt.Errorf("autostart disable: %w", err)
		}
		if jsonEnabled(cmd) {
			printJSON(autostartJSON{Enabled: false, Status: "disabled", EntryPath: m.EntryPath(), Port: 0})
			return nil
		}
		fmt.Println()
		fmt.Println("  ✓ Autostart disabled")
		fmt.Println()
		return nil
	},
}
