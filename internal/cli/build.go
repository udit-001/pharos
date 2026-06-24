package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

const buildOutput = "pharos"

var buildFlags struct {
	noCSS bool
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the pharos binary (rebuilds CSS + compiles Go)",
	Long: `Build the pharos CLI binary from source.

Steps:
  1. Rebuild CSS from web/input.css using the Tailwind CLI (unless --no-css)
  2. Copy the generated CSS into the embed directory
  3. Compile the Go binary to ./pharos

Requires 'go' on PATH and the Tailwind CLI at .bin/tailwindcss.
Run 'pharos tailwind download' first if needed.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBuild()
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().BoolVar(&buildFlags.noCSS, "no-css", false, "Skip CSS rebuild")
}

func runBuild() error {
	root := mustProjectRoot()

	if !buildFlags.noCSS {
		tailwindBin := filepath.Join(root, ".bin", "tailwindcss")
		if _, err := os.Stat(tailwindBin); err != nil {
			return fmt.Errorf("tailwind CLI not found at %s — run `pharos tailwind download` first", tailwindBin)
		}

		fmt.Println()
		fmt.Println("  Building CSS...")

		tw := exec.Command(tailwindBin,
			"--input", tailwindInput,
			"--output", tailwindOutput,
			"--content", tailwindContent,
			"--minify")
		tw.Dir = root
		tw.Stdout = os.Stdout
		tw.Stderr = os.Stderr
		if err := tw.Run(); err != nil {
			return fmt.Errorf("tailwind build failed: %w", err)
		}

		cssOut := filepath.Join(root, tailwindOutput)
		cssEmbed := filepath.Join(root, tailwindCSSEmbed)
		data, err := os.ReadFile(cssOut)
		if err != nil {
			return fmt.Errorf("read generated CSS: %w", err)
		}
		if err := os.WriteFile(cssEmbed, data, 0o644); err != nil {
			return fmt.Errorf("write embedded CSS: %w", err)
		}
	}

	fmt.Println()
	fmt.Println("  Compiling Go binary...")

	cmd := exec.Command("go", "build", "-o", buildOutput, "./cmd/pharos")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	fmt.Println()
	fmt.Println("  ✓ Built pharos")
	fmt.Println()

	return nil
}
