package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and manage pharos configuration",
	Long: `View and update pharos configuration.

The config file (pharos.toml) lives in your platform app config
directory (~/.config/pharos/ on Linux) and points to your data
directory where the database and workspaces live.

Examples:
  pharos config read             # Show current config
  pharos config set data_dir ~/my-pharos  # Change data directory`,
}

var configReadCmd = &cobra.Command{
	Use:   "read",
	Short: "Read current configuration",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("config error: %w", err)
		}

		fmt.Println()
		fmt.Printf("  Config file: %s\n", config.Path())
		if cfg != nil {
			fmt.Printf("  data_dir:    %s\n", cfg.DataDir)
			fmt.Printf("  Database:    %s\n", filepath.Join(cfg.DataDir, "pharos.db"))
		} else {
			fmt.Printf("  data_dir:    %s (default — run 'pharos init' to set up)\n", config.DefaultDataDir())
		}
		fmt.Println()
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Update a configuration key.

Supported keys:
  data_dir    Path to the pharos data directory

The value is saved to the config file. Run 'pharos config read'
to verify the change.

Examples:
  pharos config set data_dir ~/my-pharos`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("config error: %w", err)
		}
		if cfg == nil {
			return fmt.Errorf("no config found — run 'pharos init' first")
		}

		switch key {
		case "data_dir":
			cfg.DataDir = value
		default:
			return fmt.Errorf("unknown config key: %s", key)
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		// Ensure the new data directory exists
		if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
			return fmt.Errorf("create data directory: %w", err)
		}

		fmt.Println()
		fmt.Printf("  ✓ %s set to %s\n", key, cfg.DataDir)
		fmt.Printf("    Config: %s\n", config.Path())
		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configReadCmd)
	configCmd.AddCommand(configSetCmd)
}
