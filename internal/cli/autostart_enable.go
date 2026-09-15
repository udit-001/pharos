package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var autostartEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable Pharos at login (create/rewrite the startup entry)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		m, port, err := autostartManagerFor(cmd)
		if err != nil {
			return err
		}
		prev, err := m.Enable()
		if err != nil {
			return fmt.Errorf("autostart enable: %w", err)
		}
		if jsonEnabled(cmd) {
			printJSON(autostartJSON{Enabled: true, Status: "enabled", EntryPath: m.EntryPath(), Port: port})
			return nil
		}
		fmt.Println()
		fmt.Println("  ✓ Autostart enabled")
		fmt.Printf("    Entry: %s\n", m.EntryPath())
		if prev > 0 && prev != port {
			fmt.Printf("    Port:  %d (was %d)\n", port, prev)
		}
		fmt.Println()
		return nil
	},
}
