package cli

import (
	"fmt"
	"os"
	"path/filepath"

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
	Short: "List assets in the workspace",
	Long:  `List all asset files in the workspace's assets/ directory.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()

		assets, err := wsStore.ListAssets()
		if err != nil {
			return formatError("failed to list assets", err)
		}

		if jsonOut {
			type assetInfo struct {
				Name string `json:"name"`
				Path string `json:"path"`
			}
			result := make([]assetInfo, len(assets))
			for i, name := range assets {
				result[i] = assetInfo{
					Name: name,
					Path: filepath.Join(ws.Path, "assets", name),
				}
			}
			printJSON(result)
			return nil
		}

		fmt.Println()
		fmt.Printf("  Assets for %s:\n", ws.DisplayName())
		fmt.Println()
		if len(assets) == 0 {
			fmt.Println("  No assets yet.")
			fmt.Println()
			return nil
		}
		rows := make([][]string, 0, len(assets))
		for _, name := range assets {
			rows = append(rows, []string{name, filepath.Join(ws.Path, "assets", name)})
		}
		fmt.Println(formatTable([]string{"Name", "Path"}, rows))
		fmt.Println()
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
