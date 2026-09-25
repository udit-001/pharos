package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/config"
)

// ─── Exec channel (LEARN-237): run SQL in a live bench tab ──────────
//
// `pharos workbench exec` sends a run/setProblem/reset command to a
// lesson page's bench through the dashboard server and prints the reply:
// the engine Outcome verbatim plus the step verdict when a problem is
// loaded. `workbench verify` composes the verify protocol from those
// commands (setProblem → run reference → reset), so agent-authored
// problems get a mechanical PASS/FAIL against their own `test`.
//
// The exec endpoint is token-gated (config.WorkbenchTokenPath): the CLI
// reads the same 0600 token file the server generated. A random local
// page cannot execute SQL in the user's bench or forge verdicts.

type workbenchExecBody struct {
	Namespace  string                 `json:"namespace"`
	Command    map[string]interface{} `json:"command"`
	TimeoutSec int                    `json:"timeoutSec,omitempty"`
}

// postWorkbenchCommand sends one command to the exec endpoint and
// returns the parsed reply. Verbatim contract: the reply's outcome,
// verdict, and error pass through untouched.
func postWorkbenchCommand(workspace, namespace string, command map[string]interface{}, timeoutSec int) (map[string]interface{}, error) {
	port, ok := runningServerPort()
	if !ok {
		return nil, fmt.Errorf("no dashboard server running. Start it with: pharos start")
	}
	token, err := os.ReadFile(config.WorkbenchTokenPath())
	if err != nil || len(token) == 0 {
		return nil, fmt.Errorf("no workbench token at %s — start the dashboard once to generate it", config.WorkbenchTokenPath())
	}
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	body, _ := json.Marshal(workbenchExecBody{
		Namespace:  namespace,
		Command:    command,
		TimeoutSec: timeoutSec,
	})
	req, err := http.NewRequest("POST",
		"http://127.0.0.1:"+strconv.Itoa(port)+"/api/workspaces/name/"+workspacePathEscape(workspace)+"/workbench-exec",
		bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Pharos-Workbench-Token", string(token))
	// The server-side wait is timeout + relay margin; the client waits a
	// little longer so the server's error copy wins over a client abort.
	client := &http.Client{Timeout: time.Duration(timeoutSec+5) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach dashboard: %v", err)
	}
	defer resp.Body.Close()
	var parsed map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("dashboard reply was not JSON (status %d)", resp.StatusCode)
	}
	// Error bodies are {"error": "..."}; success bodies are the bench reply.
	if errStr, isErr := parsed["error"].(string); isErr {
		return nil, fmt.Errorf("%s", errStr)
	}
	return parsed, nil
}

// workspacePathEscape mirrors the relay's encodeURIComponent for the
// workspace path segment.
func workspacePathEscape(name string) string {
	var b bytes.Buffer
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '.' || r == '~' {
			b.WriteRune(r)
		} else {
			fmt.Fprintf(&b, "%%%02X", r)
		}
	}
	return b.String()
}

var workbenchExecCmd = &cobra.Command{
	Use:   "exec",
	Short: "Run a command in a live bench tab",
	Long: `Run SQL in a lesson page's bench and print the outcome.

The lesson page must be open in the dashboard (pharos nav <url>) — the
exec channel reaches the bench through the live relay. Runs journal
with actor "agent" so the practice record shows who ran what.

Examples:
  pharos workbench exec --namespace lesson-1 --sql "SELECT count(*) FROM movies"
  pharos workbench exec --namespace lesson-1 --sql "SELECT ..." --timeout 45`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace, _ := cmd.Flags().GetString("namespace")
		sql, _ := cmd.Flags().GetString("sql")
		actor, _ := cmd.Flags().GetString("actor")
		timeout, _ := cmd.Flags().GetInt("timeout")
		wsName, _ := cmd.Flags().GetString("workspace")

		if sql == "" {
			return fmt.Errorf("--sql is required")
		}
		if namespace == "" {
			return fmt.Errorf("--namespace is required — the journal namespace of the bench to command")
		}
		if actor == "" {
			actor = "agent"
		}

		wsStore, err := resolveWorkspace(mustStore(cmd), wsName)
		if err != nil {
			return err
		}
		reply, err := postWorkbenchCommand(wsStore.Workspace().Name, namespace, map[string]interface{}{
			"op": "run", "sql": sql, "actor": actor,
		}, timeout)
		if err != nil {
			return fmt.Errorf("workbench exec: %v", err)
		}
		return printWorkbenchReply(reply)
	},
}

// printWorkbenchReply renders one exec reply for an agent (or human):
// PASS/FAIL first, then the verbatim detail. One line per fact, never
// truncated — the agent parses this output.
func printWorkbenchReply(reply map[string]interface{}) error {
	outcome, _ := reply["outcome"].(map[string]interface{})
	if outcome == nil {
		return fmt.Errorf("reply carried no outcome")
	}
	if verdict, hasVerdict := reply["verdict"].(map[string]interface{}); hasVerdict {
		if verdict["outcome"] == "pass" {
			fmt.Println("  PASS — attempt matched the problem's test")
		} else {
			fmt.Println("  FAIL — attempt missed the problem's test")
			if detail, ok := verdict["detail"].(string); ok && detail != "" {
				fmt.Println("  " + detail)
			}
		}
	} else {
		fmt.Println("  Ran (no problem loaded — no verdict)")
	}
	// The verbatim outcome underneath the verdict line.
	outcomeJSON, _ := json.Marshal(outcome)
	fmt.Printf("  outcome: %s\n", outcomeJSON)
	return nil
}

var workbenchVerifyCmd = &cobra.Command{
	Use:   "verify <problem.json> --sql <reference.sql>",
	Short: "Verify a problem file's test against its reference solution",
	Long: `Compose the verify protocol against a live bench tab (LEARN-236):

  setProblem(problem) → run(reference SQL) → reset

PASS requires the reference solution to match the problem's own test —
an author-hallucinated test fails here, before any learner sees it.
Always finishes with reset so the lesson page stays clean for learners.

The problem JSON shape:
  { "label": "...", "concept": "...", "test": { "rows": [["x", 1]] } }
  --sql is the path to the reference solution file.

Examples:
  pharos workbench verify ./revenue-top5.json --sql ./reference.sql --namespace lesson-1`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		problemPath := args[0]
		sqlFile, _ := cmd.Flags().GetString("sql")
		namespace, _ := cmd.Flags().GetString("namespace")
		timeout, _ := cmd.Flags().GetInt("timeout")
		wsName, _ := cmd.Flags().GetString("workspace")

		if namespace == "" {
			return fmt.Errorf("--namespace is required")
		}
		if sqlFile == "" {
			return fmt.Errorf("--sql is required (path to the reference solution)")
		}
		reference, err := os.ReadFile(sqlFile)
		if err != nil {
			return fmt.Errorf("failed to read reference SQL: %v", err)
		}
		problemRaw, err := os.ReadFile(problemPath)
		if err != nil {
			return fmt.Errorf("failed to read problem file: %v", err)
		}
		var problemObj map[string]interface{}
		if err := json.Unmarshal(problemRaw, &problemObj); err != nil {
			return fmt.Errorf("problem file is not valid JSON: %v", err)
		}
		if _, ok := problemObj["test"]; !ok {
			return fmt.Errorf(`problem file needs a "test" object`)
		}
		// The bench knows the Problem shape; authoring metadata the slot
		// doesn't declare is stripped, not forwarded.
		delete(problemObj, "reference")

		wsStore, err := resolveWorkspace(mustStore(cmd), wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace().Name

		// 1. Load the problem onto the slot.
		if _, err := postWorkbenchCommand(ws, namespace, map[string]interface{}{
			"op": "setProblem", "problem": problemObj,
		}, timeout); err != nil {
			return fmt.Errorf("verify: %v", err)
		}

		// 2. Run the reference solution.
		reply, err := postWorkbenchCommand(ws, namespace, map[string]interface{}{
			"op": "run", "sql": string(reference), "actor": "agent",
		}, timeout)
		if err != nil {
			if _, resetErr := postWorkbenchCommand(ws, namespace, map[string]interface{}{"op": "reset"}, timeout); resetErr != nil {
				fmt.Fprintf(os.Stderr, "  warning: reset failed: %v\n", resetErr)
			}
			return fmt.Errorf("verify: %v", err)
		}
		pass := false
		if verdict, ok := reply["verdict"].(map[string]interface{}); ok {
			pass = verdict["outcome"] == "pass"
		}
		_ = printWorkbenchReply(reply)

		// 3. Restore clean state for learners — regardless of verdict.
		if _, resetErr := postWorkbenchCommand(ws, namespace, map[string]interface{}{"op": "reset"}, timeout); resetErr != nil {
			fmt.Fprintf(os.Stderr, "  warning: reset failed: %v\n", resetErr)
		}

		if !pass {
			os.Exit(1)
		}
		fmt.Println("  ✓ verified — if this was not expected, regenerate the test from actual output")
		return nil
	},
}

func init() {
	workbenchExecCmd.Flags().String("sql", "", "SQL to run in the live bench")
	workbenchExecCmd.Flags().String("namespace", "", "the bench's journal namespace")
	workbenchExecCmd.Flags().String("actor", "agent", "journal actor: agent (default) or learner")
	workbenchExecCmd.Flags().Int("timeout", 30, "seconds to wait for the bench reply")
	workbenchVerifyCmd.Flags().String("sql", "", "path to the reference solution SQL file")
	workbenchVerifyCmd.Flags().String("namespace", "", "the lesson page's bench namespace")
	workbenchVerifyCmd.Flags().Int("timeout", 30, "seconds to wait per command")
	workbenchCmd.AddCommand(workbenchExecCmd)
	workbenchCmd.AddCommand(workbenchVerifyCmd)
}
