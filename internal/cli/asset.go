package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
)

var assetCmd = &cobra.Command{
	Use:   "asset",
	Short: "Manage workspace assets",
	Long: `Manage reusable components (stylesheets, scripts, images) in the
workspace's assets/ directory.

Assets are raw files with no database tracking — they're referenced
by lessons and references via root-relative URLs (assets/style.css).

Examples:
  pharos asset list --workspace "sql-for-research"
  pharos asset create style.css --workspace "sql-for-research" --body-file /tmp/style.css`,
}

var assetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workspace assets and available vendored assets",
	Long: `List all assets in the workspace's assets/ directory, plus known
vendored assets (mermaid, highlightjs) and whether they are present.

Use 'pharos asset add <name>' to download a vendored asset.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()

		present, err := wsStore.ListAssets()
		if err != nil {
			return formatError("failed to list assets", err)
		}
		presentSet := make(map[string]bool, len(present))
		for _, f := range present {
			presentSet[f] = true
		}

		// Registry entries sorted by name
		names := make([]string, 0, len(knownAssets))
		for k := range knownAssets {
			names = append(names, k)
		}
		sort.Strings(names)

		type regEntry struct {
			Name     string `json:"name"`
			Filename string `json:"filename"`
			Present  bool   `json:"present"`
			Version  string `json:"default_version"`
		}
		registry := make([]regEntry, 0, len(names))
		covered := make(map[string]bool, len(names))
		for _, n := range names {
			spec := knownAssets[n]
			_, found := presentSet[spec.Filename]
			registry = append(registry, regEntry{n, spec.Filename, found, spec.DefaultVersion})
			covered[spec.Filename] = true
		}

		// Files in assets/ that aren't from the registry
		extras := make([]string, 0, len(present))
		for _, f := range present {
			if !covered[f] {
				extras = append(extras, f)
			}
		}
		sort.Strings(extras)

		if jsonOut {
			printJSON(map[string]any{
				"vendored": registry,
				"user":     extras,
			})
			return nil
		}

		fmt.Println()
		fmt.Printf("  Assets for %s:\n", ws.DisplayName())
		fmt.Println()

		regRows := make([][]string, 0, len(registry))
		for _, r := range registry {
			status := "absent"
			if r.Present {
				status = "present"
			}
			regRows = append(regRows, []string{r.Name, status, "pharos asset add " + r.Name})
		}
		fmt.Println("  Vendored assets:")
		fmt.Print(formatTable([]string{"Name", "Status", "Add command"}, regRows))
		fmt.Println()

		if len(extras) > 0 {
			fmt.Println("  Other assets:")
			extraRows := make([][]string, 0, len(extras))
			for _, f := range extras {
				extraRows = append(extraRows, []string{f, filepath.Join("assets", f)})
			}
			fmt.Print(formatTable([]string{"Name", "Path"}, extraRows))
			fmt.Println()
		}

		if len(registry)+len(extras) == 0 {
			fmt.Println("  No assets yet.")
			fmt.Println()
		}
		return nil
	},
}

var assetCreateCmd = &cobra.Command{
	Use:   "create <filename>",
	Short: "Create or overwrite an asset file",
	Long: `Write a file to the workspace's assets/ directory. Overwrites if it exists.

Examples:
  pharos asset create style.css --workspace "sql-for-research" --body-file /tmp/style.css
  pharos asset create quiz-widget.js --workspace "yoga" --body-file /tmp/widget.js`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		filename := args[0]
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()

		bodyFile, _ := cmd.Flags().GetString("body-file")
		if bodyFile == "" {
			return fmt.Errorf("--body-file is required\n  pharos asset create %q --workspace %q --body-file <path>", filename, ws.Name)
		}

		data, err := os.ReadFile(bodyFile)
		if err != nil {
			return fmt.Errorf("read body file: %w", err)
		}

		if err := wsStore.CreateAsset(filename, string(data)); err != nil {
			return fmt.Errorf("create asset: %w", err)
		}

		assetPath := filepath.Join(ws.Path, "assets", filename)

		if jsonOut {
			type assetResult struct {
				Created   bool   `json:"created"`
				Filename  string `json:"filename"`
				Path      string `json:"path"`
				Workspace string `json:"workspace"`
			}
			printJSON(assetResult{
				Created:   true,
				Filename:  filename,
				Path:      assetPath,
				Workspace: ws.Name,
			})
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Asset created: %s\n", filename)
		fmt.Printf("    File: %s\n", assetPath)
		fmt.Printf("    Workspace: %s\n", ws.DisplayName())
		fmt.Println()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(assetCmd)
	assetCmd.AddCommand(assetListCmd)
	assetCmd.AddCommand(assetCreateCmd)
	assetListCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	assetCreateCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	assetCreateCmd.Flags().String("body-file", "", "Read asset content from a file (required)")
}
