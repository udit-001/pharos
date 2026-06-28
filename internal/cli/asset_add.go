package cli

import (
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type assetSpec struct {
	Filename       string
	DefaultVersion string
	URLTemplate    string            // {{VERSION}} placeholder
	ExtraFiles     map[string]string // filename → content (generated alongside download)
}

//go:embed highlight.css
var highlightCSS string

//go:embed mermaid-lightbox.css
var mermaidLightboxCSS string

//go:embed mermaid-lightbox.js
var mermaidLightboxJS string

//go:embed mermaid-theme.js
var mermaidThemeJS string

var knownAssets = map[string]assetSpec{
	"mermaid": {
		Filename:       "mermaid.min.js",
		DefaultVersion: "11",
		URLTemplate:    "https://cdn.jsdelivr.net/npm/mermaid@{{VERSION}}/dist/mermaid.min.js",
		ExtraFiles: map[string]string{
			"mermaid-theme.js": mermaidThemeJS,
		},
	},
	"mermaid-lightbox": {
		Filename:       "",
		DefaultVersion: "",
		URLTemplate:    "",
		ExtraFiles: map[string]string{
			"mermaid-lightbox.js":  mermaidLightboxJS,
			"mermaid-lightbox.css": mermaidLightboxCSS,
		},
	},
	"highlightjs": {
		Filename:       "highlight.min.js",
		DefaultVersion: "11.11.1",
		URLTemplate:    "https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@{{VERSION}}/build/highlight.min.js",
		ExtraFiles: map[string]string{
			"highlight.css": highlightCSS,
		},
	},
}

var assetAddCmd = &cobra.Command{
	Use:   "add <name>[@version]",
	Short: "Download a vendored asset to the workspace",
	Long: `Download a known vendored asset into the workspace's assets/ directory.

Use 'pharos asset list' to see available assets and which are already present.

Versions can be pinned via the @ suffix (mermaid@11). Omit for the default.

Examples:
  pharos asset add mermaid
  pharos asset add mermaid@10
  pharos asset add highlightjs`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()

		name, version := parseAssetVersion(args[0])
		spec, ok := knownAssets[name]
		if !ok {
			return fmt.Errorf("unknown asset %q\n  Available: %s", name, knownAssetsString())
		}

		if version == "" {
			version = spec.DefaultVersion
		}

		var targetPath string
		if spec.Filename != "" {
			targetPath = filepath.Join(ws.Path, "assets", spec.Filename)

			if _, err := os.Stat(targetPath); err == nil && spec.URLTemplate != "" {
				// downloaded asset already present — skip
				if jsonOut {
					printJSON(map[string]any{
						"present":  true,
						"name":     name,
						"version":  version,
						"filename": spec.Filename,
					})
					return nil
				}
				fmt.Println()
				fmt.Printf("  ✓ %s@%s is already present — skipping\n", name, version)
				fmt.Println()
				return nil
			}

			if spec.URLTemplate != "" {
				if version == "" {
					version = spec.DefaultVersion
				}
				url := strings.ReplaceAll(spec.URLTemplate, "{{VERSION}}", version)

				resp, err := http.Get(url)
				if err != nil {
					return fmt.Errorf("download %s: %w", name, err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					return fmt.Errorf("download %s: server returned %s", name, resp.Status)
				}

				data, err := io.ReadAll(resp.Body)
				if err != nil {
					return fmt.Errorf("read response: %w", err)
				}

				if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
					return fmt.Errorf("create assets dir: %w", err)
				}
				if err := os.WriteFile(targetPath, data, 0o644); err != nil {
					return fmt.Errorf("write asset: %w", err)
				}
			}
		}

		var extras []string
		for fname, content := range spec.ExtraFiles {
			fp := filepath.Join(ws.Path, "assets", fname)
			if _, err := os.Stat(fp); err != nil {
				if err := os.WriteFile(fp, []byte(content), 0o644); err != nil {
					return fmt.Errorf("write %s: %w", fname, err)
				}
				extras = append(extras, fname)
			}
		}

		_ = wsStore.Touch()

		if jsonOut {
			extraPaths := make([]string, len(extras))
			for i, f := range extras {
				extraPaths[i] = filepath.Join(ws.Path, "assets", f)
			}
			printJSON(map[string]any{
				"present":     false,
				"name":        name,
				"version":     version,
				"filename":    spec.Filename,
				"path":        targetPath,
				"extra_files": extraPaths,
			})
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Added %s\n", name)
		if targetPath != "" {
			fmt.Printf("    File: %s\n", targetPath)
		}
		for _, f := range extras {
			fmt.Printf("    File: %s\n", filepath.Join(ws.Path, "assets", f))
		}
		fmt.Println()
		return nil
	},
}

func knownAssetsString() string {
	names := make([]string, 0, len(knownAssets))
	for k := range knownAssets {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func parseAssetVersion(input string) (name, version string) {
	if idx := strings.LastIndex(input, "@"); idx > 0 {
		return input[:idx], input[idx+1:]
	}
	return input, ""
}

func init() {
	assetCmd.AddCommand(assetAddCmd)
	assetAddCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
