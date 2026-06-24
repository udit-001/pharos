package cli

import (
	"github.com/spf13/cobra"
)

const (
	tailwindInput    = "web/input.css"
	tailwindOutput   = "web/app.css"
	tailwindContent  = "**/*.go"
	tailwindCSSEmbed = "internal/web/app.css"
)

var tailwindCmd = &cobra.Command{
	Use:   "tailwind",
	Short: "Manage the Tailwind CSS CLI binary",
	Long: `Download and manage the Tailwind CSS standalone CLI binary.

The Tailwind CSS standalone CLI is required for CSS compilation.
Use 'pharos tailwind download' to download it for your platform.`,
}

func init() {
	rootCmd.AddCommand(tailwindCmd)
}
