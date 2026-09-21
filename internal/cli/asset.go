package cli

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
)

var assetCmd = &cobra.Command{
	Use:   "asset",
	Short: "Manage workspace assets",
	Args:  cobra.NoArgs,
	RunE:  runShowHelp,
	Long: `Manage reusable components (stylesheets, scripts, images) in the
workspace's assets/ directory.

Assets are user-authored files with no database tracking — they're
referenced by lessons and references via root-relative URLs
(assets/style.css). Third-party libraries are not assets: they live in
the global vendor cache (see 'pharos vendor sync').

Examples:
  pharos asset list --workspace "sql-for-research"
  pharos asset create style.css --workspace "sql-for-research" --body-file /tmp/style.css`,
}

var assetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the workspace's user components",
	Long: `List files in the workspace's assets/ directory. These are user
components — stylesheets, scripts, images authored with
'pharos asset create'. Vendored third-party libraries are not listed
here; they live in the global vendor cache (see 'pharos vendor sync').`,
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
		sort.Strings(present)

		if jsonEnabled(cmd) {
			printJSON(map[string]any{"assets": present})
			return nil
		}

		fmt.Println()
		fmt.Printf("  Assets for %s:\n", ws.DisplayName())
		fmt.Println()

		if len(present) > 0 {
			rows := make([][]string, 0, len(present))
			for _, f := range present {
				rows = append(rows, []string{f, filepath.Join("assets", f)})
			}
			fmt.Print(formatTable([]string{"Name", "Path"}, rows))
			fmt.Println()
		} else {
			fmt.Printf("  No assets yet.\n")
			fmt.Printf("  Use 'pharos asset create <filename> --body-file <path>' to add one.\n")
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

		data, err := readBodyFile(bodyFile)
		if err != nil {
			return err
		}

		if err := wsStore.WriteAsset(filename, data); err != nil {
			return fmt.Errorf("create asset: %w", err)
		}

		assetPath := filepath.Join(ws.Path, "assets", filename)

		if jsonEnabled(cmd) {
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
