package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
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
	Short: "List workspace assets (seeded, vendored, user)",
	Long: `List assets in the workspace's assets/ directory, grouped by source:

  Seeded   — universal defaults every workspace starts with (style.css,
             glossary-tooltip.js, copy-code.js, the Inter font).
  Vendored — third-party libraries added on demand (mermaid, highlightjs,
             mermaid-lightbox).
  User     — components authored with 'pharos asset create'.

Each row shows whether the asset is fully present and the command that acts
on it: 'pharos asset add' when absent, 'pharos asset redeploy' when present.`,
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

		// Registry names sorted, split by source.
		names := make([]string, 0, len(knownAssets))
		for k := range knownAssets {
			names = append(names, k)
		}
		sort.Strings(names)

		// owned = every file the registry claims (lib Filename + embedded
		// Files), so present files left over are user-authored.
		owned := make(map[string]bool)
		for _, spec := range knownAssets {
			if spec.Filename != "" {
				owned[spec.Filename] = true
			}
			for f := range spec.Files {
				owned[f] = true
			}
		}

		seeded := make([]regEntry, 0, len(names))
		vendored := make([]regEntry, 0, len(names))
		for _, n := range names {
			spec := knownAssets[n]
			p := specPresent(spec, presentSet)
			hint := "pharos asset redeploy " + n
			if !p {
				hint = "pharos asset add " + n
			}
			entry := regEntry{Name: n, Present: p, Hint: hint}
			if spec.Source == "seeded" {
				seeded = append(seeded, entry)
			} else {
				vendored = append(vendored, entry)
			}
		}

		user := make([]string, 0, len(present))
		for _, f := range present {
			if !owned[f] {
				user = append(user, f)
			}
		}
		sort.Strings(user)

		if jsonOut {
			printJSON(map[string]any{
				"seeded":   seeded,
				"vendored": vendored,
				"user":     user,
			})
			return nil
		}

		fmt.Println()
		fmt.Printf("  Assets for %s:\n", ws.DisplayName())
		fmt.Println()

		fmt.Println("  Seeded assets:")
		fmt.Print(formatTable([]string{"Name", "Status", "Command"}, regRows(seeded)))
		fmt.Println()

		fmt.Println("  Vendored assets:")
		fmt.Print(formatTable([]string{"Name", "Status", "Command"}, regRows(vendored)))
		fmt.Println()

		if len(user) > 0 {
			fmt.Println("  User assets:")
			userRows := make([][]string, 0, len(user))
			for _, f := range user {
				userRows = append(userRows, []string{f, filepath.Join("assets", f)})
			}
			fmt.Print(formatTable([]string{"Name", "Path"}, userRows))
			fmt.Println()
		}

		if len(seeded)+len(vendored)+len(user) == 0 {
			fmt.Println("  No assets yet.")
			fmt.Println()
		}
		return nil
	},
}

// regEntry is a registry asset row in the list output (seeded or vendored).
type regEntry struct {
	Name    string `json:"name"`
	Present bool   `json:"present"`
	Hint    string `json:"hint"`
}

// specPresent reports whether every file a spec owns (its lib Filename plus
// all embedded Files) is present on disk.
func specPresent(spec db.AssetSpec, presentSet map[string]bool) bool {
	if spec.Filename != "" && !presentSet[spec.Filename] {
		return false
	}
	for f := range spec.Files {
		if !presentSet[f] {
			return false
		}
	}
	return true
}

// regRows builds table rows for a slice of registry entries.
func regRows(entries []regEntry) [][]string {
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		status := "absent"
		if e.Present {
			status = "present"
		}
		rows = append(rows, []string{e.Name, status, e.Hint})
	}
	return rows
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

		if err := wsStore.WriteAsset(filename, data); err != nil {
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
