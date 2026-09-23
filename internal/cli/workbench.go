package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
	"github.com/udit-001/pharos/internal/workbench"
)

var workbenchCmd = &cobra.Command{
	Use:   "workbench",
	Short: "sql-workbench event log, datasets, and validation",
	Long: `Manage sql-workbench browser events and datasets.

The workbench logs browser events (query, error, info, dataset) from the
sql-workbench component. Events are ingested via POST and queried via CLI.

CLI vocabulary uses model words ("dataset", "journal", "namespace").
Learner words ("sample data", "history") are UI-only.

Examples:
  pharos workbench log
  pharos workbench log --limit 50 --type query
  pharos workbench log --namespace bench --json
  pharos workbench check ./my-dataset.json
  pharos workbench add ./my-dataset.json
  pharos workbench remove my-dataset`,
	Args: cobra.NoArgs,
	RunE: runShowHelp,
}

var workbenchLogCmd = &cobra.Command{
	Use:   "log",
	Short: "Show recent workbench events",
	Long: `Show recent workbench events for the current workspace, newest first.

Events from all namespaces are merged by default. Filter with --namespace
or --type. The output is one line per event, never truncated — the agent
reads the full log.

Examples:
  pharos workbench log
  pharos workbench log --limit 100
  pharos workbench log --type query
  pharos workbench log --namespace bench
  pharos workbench log --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		ns, _ := cmd.Flags().GetString("namespace")
		evType, _ := cmd.Flags().GetString("type")

		events, err := wsStore.GetWorkbenchEvents(ns, evType, limit)
		if err != nil {
			return formatError("failed to read workbench events", err)
		}

		if len(events) == 0 {
			fmt.Println("  no workbench events yet")
			return nil
		}

		if jsonEnabled(cmd) {
			for _, ev := range events {
				// One JSON object per line (NDJSON).
				m := map[string]string{
					"id":        ev.ID,
					"namespace": ev.Namespace,
					"type":      ev.Type,
					"ts":        ev.Ts,
				}
				// Merge payload fields at top level.
				var p map[string]any
				if ev.Payload != "" {
					if err := json.Unmarshal([]byte(ev.Payload), &p); err == nil {
						for k, v := range p {
							m[k] = fmt.Sprintf("%v", v)
						}
					} else {
						m["payload"] = ev.Payload
					}
				}
				b, _ := json.Marshal(m)
				fmt.Println(string(b))
			}
			return nil
		}

		// Table output: one line per event. The payload is a single-line JSON
		// blob (newlines are escaped inside strings), so printing it verbatim
		// keeps the one-line shape — and honors the verbatim contract
		// (LEARN-206 decision 6): long SQL and error text must survive intact.
		fmt.Println()
		for _, ev := range events {
			fmt.Printf("  %s  %-10s  %s\n", ev.Ts, ev.Type, strings.TrimSpace(ev.Payload))
		}
		fmt.Println()
		return nil
	},
}

var workbenchCheckCmd = &cobra.Command{
	Use:   "check <file.json>",
	Short: "Validate a workbench event or dataset file",
	Long: `Validate a JSON file containing workbench events or a dataset.

For event files: validates the shape of each event (type, required fields)
and reports any errors verbatim. Exit code 1 on validation failure.

For dataset files: validates the JSON structure and filename format.

Examples:
  pharos workbench check ./events.json
  pharos workbench check ./my-dataset.json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		data, err := os.ReadFile(filePath)
		if err != nil {
			return formatError("failed to read file", err)
		}

		// Try parsing as an event batch first.
		var batch struct {
			Events []workbench.WorkbenchEvent `json:"events"`
		}
		if err := json.Unmarshal(data, &batch); err == nil && len(batch.Events) > 0 {
			// Validate as event batch.
			var errs []string
			for i, ev := range batch.Events {
				if err := workbench.ValidateEvent(ev); err != nil {
					errs = append(errs, fmt.Sprintf("event[%d]: %s", i, err))
				}
			}
			if len(errs) > 0 {
				return fmt.Errorf("%s", strings.Join(errs, "\n"))
			}
			fmt.Printf("  ✓ %d events valid\n", len(batch.Events))
			return nil
		}

		// Try parsing as a single event.
		var ev workbench.WorkbenchEvent
		if err := json.Unmarshal(data, &ev); err == nil && ev.ID != "" {
			if err := workbench.ValidateEvent(ev); err != nil {
				return err
			}
			fmt.Println("  ✓ event valid")
			return nil
		}

		// Try as a dataset file.
		id := datasetIDFromFilename(filePath)
		if err := workbench.ValidateDatasetUpload(id, data); err != nil {
			return err
		}
		fmt.Printf("  ✓ dataset %q valid\n", id)
		return nil
	},
}

var workbenchAddCmd = &cobra.Command{
	Use:   "add <file.json>",
	Short: "Validate and install a dataset into the workspace",
	Long: `Validate a dataset file (same as 'workbench check'), then install it
into the workspace's datasets/ directory. The filename stem becomes the
dataset id. Fails if validation fails.

Examples:
  pharos workbench add ./my-data.json
  pharos workbench add ./samples/quiz-data.json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}

		filePath := args[0]
		data, err := os.ReadFile(filePath)
		if err != nil {
			return formatError("failed to read file", err)
		}

		id := datasetIDFromFilename(filePath)
		if err := workbench.ValidateDatasetUpload(id, data); err != nil {
			return err
		}

		ws := wsStore.Workspace()
		datasetsDir := filepath.Join(ws.Path, "datasets")
		if err := os.MkdirAll(datasetsDir, 0755); err != nil {
			return formatError("failed to create datasets directory", err)
		}

		dest := filepath.Join(datasetsDir, id+".json")
		if err := os.WriteFile(dest, data, 0644); err != nil {
			return formatError("failed to write dataset", err)
		}

		if jsonEnabled(cmd) {
			printJSON(map[string]string{"id": id, "path": dest})
		} else {
			fmt.Printf("  ✓ dataset %q installed\n", id)
			fmt.Printf("    File: %s\n", dest)
		}
		return nil
	},
}

var workbenchRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove a dataset from the workspace",
	Long: `Remove a dataset by its id (filename stem) from the workspace's
datasets/ directory. The file must exist; removal is not idempotent.

Examples:
  pharos workbench remove my-data
  pharos workbench remove quiz-dataset`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}

		id := args[0]
		if err := workbench.ValidateDataset(id); err != nil {
			return err
		}

		ws := wsStore.Workspace()
		filePath := filepath.Join(ws.Path, "datasets", id+".json")

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("dataset %q not found", id)
		}

		if err := os.Remove(filePath); err != nil {
			return formatError("failed to remove dataset", err)
		}

		if jsonEnabled(cmd) {
			printJSON(map[string]string{"id": id})
		} else {
			fmt.Printf("  ✓ dataset %q removed\n", id)
		}
		return nil
	},
}

func init() {
	workbenchLogCmd.Flags().Int("limit", 200, "Maximum events to show")
	workbenchLogCmd.Flags().String("namespace", "", "Filter by namespace")
	workbenchLogCmd.Flags().String("type", "", "Filter by event type (query, error, info, dataset)")
	workbenchLogCmd.Flags().StringP("workspace", "w", "", "Workspace name")

	workbenchAddCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	workbenchRemoveCmd.Flags().StringP("workspace", "w", "", "Workspace name")

	workbenchCmd.AddCommand(workbenchLogCmd)
	workbenchCmd.AddCommand(workbenchCheckCmd)
	workbenchCmd.AddCommand(workbenchAddCmd)
	workbenchCmd.AddCommand(workbenchRemoveCmd)

	rootCmd.AddCommand(workbenchCmd)
}

// datasetIDFromFilename extracts the dataset id (filename stem) from a path,
// falling back to the full basename without extension.
func datasetIDFromFilename(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	id := strings.TrimSuffix(base, ext)
	// Normalize: lowercase, replace spaces and underscores with hyphens.
	id = strings.ToLower(id)
	id = strings.ReplaceAll(id, "_", "-")
	id = strings.ReplaceAll(id, " ", "-")
	return id
}

// Ensure unused imports don't break the build.
var _ = db.Workspace{}
