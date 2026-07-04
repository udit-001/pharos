package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/config"
	"github.com/udit-001/pharos/internal/db"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and manage pharos configuration",
	Args:  cobra.NoArgs,
	RunE:  runShowHelp,
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

		dataDir := config.DefaultDataDir()
		if cfg != nil && cfg.DataDir != "" {
			dataDir = cfg.DataDir
		}
		dbPath := filepath.Join(dataDir, "pharos.db")

		port := config.DefaultPort
		portLabel := "9090 (default)"
		if cfg != nil && cfg.Port != 0 {
			port = cfg.Port
			portLabel = fmt.Sprintf("%d", port)
		}

		fmt.Println()
		fmt.Printf("  Config file:   %s\n", config.Path())
		fmt.Printf("  data_dir:      %s\n", dataDir)
		fmt.Printf("  Database:      %s\n", dbPath)
		fmt.Printf("  port:          %s\n", portLabel)

		// Show DB-backed settings
		s, err := db.Open(dbPath)
		if err == nil {
			settings, err := s.GetSettings()
			if err == nil {
				fmt.Printf("  auto_submit:   %v\n", settings.AutoSubmitChoice)
			}
			s.Close()
		}
		fmt.Println()
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Update a configuration key or DB-backed setting.

Supported keys:
  data_dir            Path to the pharos data directory
  auto_submit_choice  Auto-submit choice questions on selection (on|off)
  port                HTTP server port for the web dashboard (1-65535)

TOML keys are saved to the config file. DB-backed settings are
persisted immediately. Run 'pharos config read' to verify.

Examples:
  pharos config set data_dir ~/my-pharos
  pharos config set auto_submit_choice on
  pharos config set port 8080`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		switch key {
		case "auto_submit_choice":
			var v bool
			switch value {
			case "on", "true", "1":
				v = true
			case "off", "false", "0":
				v = false
			default:
				return fmt.Errorf("invalid value %q for %s: use on or off", value, key)
			}
			dataDir := resolveDataDir()
			dbPath := filepath.Join(dataDir, "pharos.db")
			s, err := db.Open(dbPath)
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer s.Close()
			if err := s.SetAutoSubmitChoice(v); err != nil {
				return fmt.Errorf("set %s: %w", key, err)
			}
			fmt.Println()
			fmt.Printf("  ✓ %s set to %v\n", key, v)
			fmt.Println()
			return nil
		}

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
		case "port":
			p, err := strconv.Atoi(value)
			if err != nil || p < 1 || p > 65535 {
				return fmt.Errorf("invalid value %q for %s: use a port number 1-65535", value, key)
			}
			cfg.Port = p
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
		fmt.Printf("  ✓ %s set to %s\n", key, value)
		fmt.Printf("    Config: %s\n", config.Path())
		if key == "port" {
			fmt.Println("    Note: if you've pinned Pharos, re-create the shortcut so it points at the new port.")
		}
		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configReadCmd)
	configCmd.AddCommand(configSetCmd)
}
