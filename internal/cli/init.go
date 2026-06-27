package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/config"
	"github.com/udit-001/pharos/internal/db"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize pharos: create config and database",
	Long: `Set up pharos on this machine.

Creates the config file (pharos.toml) and SQLite database in your
data directory (default ~/.pharos).

This command is idempotent: if everything already exists it is left
untouched (use --force to recreate the database). Workspaces are
created separately with 'pharos workspace create <name>'.

Examples:
  pharos init                # Set up pharos
  pharos init --force        # Recreate the database from scratch`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		// Step 1: ensure config file exists
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("config error: %w", err)
		}
		if cfg == nil {
			cfg = &config.Config{DataDir: config.DefaultDataDir()}
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("create config: %w", err)
			}
			fmt.Printf("  ✓ Created config: %s\n", config.Path())
		} else {
			fmt.Printf("  • Config exists: %s\n", config.Path())
		}

		// Step 2: ensure data directory and database exist
		dbPath := filepath.Join(cfg.DataDir, "pharos.db")
		if _, statErr := os.Stat(dbPath); statErr == nil {
			if force {
				for _, suffix := range []string{"", "-wal", "-shm"} {
					if err := os.Remove(dbPath + suffix); err != nil && !os.IsNotExist(err) {
						return fmt.Errorf("remove existing database: %w", err)
					}
				}
				s, err := db.Open(dbPath)
				if err != nil {
					return fmt.Errorf("create database: %w", err)
				}
				_ = s.Close()
				fmt.Printf("  ✓ Recreated database: %s\n", dbPath)
			} else {
				fmt.Printf("  • Database already initialized: %s\n", dbPath)
			}
		} else {
			s, err := db.Open(dbPath)
			if err != nil {
				return fmt.Errorf("create database: %w", err)
			}
			_ = s.Close()
			fmt.Printf("  ✓ Created database: %s\n", dbPath)
		}

		fmt.Println()
		offerSkillInstall()
		fmt.Println()
		fmt.Println("  Next steps:")
		fmt.Println("    pharos workspace create \"Your topic\"")
		fmt.Println()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().Bool("force", false, "Recreate the database if it exists")
}
