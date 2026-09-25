package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/udit-001/pharos/internal/config"
	"github.com/udit-001/pharos/internal/db"
)

// ─── Exec channel: agent runs SQL in a live bench tab (LEARN-237) ────
//
// The learner's bench is a browser tab; the agent verifies problems in
// that same runtime. Three pieces, all on existing seams:
//
//   POST .../workbench-exec      CLI → server → broadcast a command over
//                                the SSE topic "workbench:<workspace>",
//                                park until the relay replies, return
//                                the reply verbatim.
//   GET  /api/events             the existing SSE broker; the relay
//                                subscribes with topic=workbench:<ws>.
//   POST .../workbench-replies   the relay posts {id, reply} when the
//                                bench finished the command.
//
// Both POSTs carry the workbench token (config.WorkbenchTokenPath) —
// the two directions where a forged or unauthorized caller could
// execute SQL in the user's tab or fake a verify verdict. Commands
// serialize downstream in the relay's own queue: the human's in-flight
// run is never preempted.

// execHub correlates a broadcast command with its relay reply. One
// parked entry per in-flight command.
type execHub struct {
	mu      sync.Mutex
	waiting map[string]chan []byte
}

func newExecHub() *execHub {
	return &execHub{waiting: make(map[string]chan []byte)}
}

func (h *execHub) register(id string) chan []byte {
	ch := make(chan []byte, 1)
	h.mu.Lock()
	h.waiting[id] = ch
	h.mu.Unlock()
	return ch
}

// resolve delivers a reply; false = expired or unknown command id
// (the relay drops those loudly at the endpoint).
func (h *execHub) resolve(id string, reply []byte) bool {
	h.mu.Lock()
	ch, ok := h.waiting[id]
	if ok {
		delete(h.waiting, id)
	}
	h.mu.Unlock()
	if !ok {
		return false
	}
	ch <- reply
	return true
}

func (h *execHub) remove(id string) {
	h.mu.Lock()
	delete(h.waiting, id)
	h.mu.Unlock()
}

// workbenchCommand is the exec request body: one bench command plus
// delivery parameters. The command shape mirrors the bench's
// bench-kit/commands.ts contract (LEARN-236): run | setProblem | reset.
type workbenchCommand struct {
	ID      string          `json:"id"`
	Op      string          `json:"op"`
	Sql     string          `json:"sql,omitempty"`
	Actor   string          `json:"actor,omitempty"`
	Problem json.RawMessage `json:"problem,omitempty"`
}

type workbenchExecRequest struct {
	Namespace  string           `json:"namespace"`
	Command    workbenchCommand `json:"command"`
	TimeoutSec int              `json:"timeoutSec,omitempty"`
}

// handleWorkbenchExec broadcasts a command to the workspace's live bench
// tabs and parks until the relay replies. No subscriber → 409 with copy
// that names the fix; timeout → 504, never a fabricated result.
func handleWorkbenchExec(store *db.Store, broker *Broker, hub *execHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkWorkbenchToken(r) {
			jsonError(w, "missing or wrong X-Pharos-Workbench-Token header — read it from "+config.WorkbenchTokenPath(), 401)
			return
		}

		name := r.PathValue("name")
		if _, err := store.Workspace(name); err != nil {
			jsonError(w, "workspace not found", 404)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			jsonError(w, "failed to read request body", 400)
			return
		}
		var req workbenchExecRequest
		if err := json.Unmarshal(body, &req); err != nil {
			jsonError(w, "request body is not valid JSON", 400)
			return
		}
		if req.Namespace == "" {
			jsonError(w, "namespace is required", 400)
			return
		}
		if len(req.Namespace) > 128 || req.Namespace != strings.TrimSpace(req.Namespace) {
			jsonError(w, "namespace must be non-empty, trimmed, and at most 128 characters", 400)
			return
		}
		switch req.Command.Op {
		case "run":
			if req.Command.Sql == "" {
				jsonError(w, "run command needs sql", 400)
				return
			}
		case "setProblem", "reset":
		default:
			jsonError(w, `unknown command op (valid: run, setProblem, reset)`, 400)
			return
		}

		timeout := time.Duration(req.TimeoutSec) * time.Second
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		if timeout > 120*time.Second {
			jsonError(w, "timeoutSec exceeds maximum of 120", 400)
			return
		}
		if req.Command.ID == "" {
			req.Command.ID = fmt.Sprintf("cmd-%d", time.Now().UnixNano())
		}

		cmdJSON, err := json.Marshal(req.Command)
		if err != nil {
			jsonError(w, "command is not encodable", 400)
			return
		}
		topic := "workbench:" + name
		delivered := broker.Broadcast(topic, Event{Type: "workbench-command", Data: cmdJSON})
		if delivered == 0 {
			jsonError(w, fmt.Sprintf(
				"no live bench tab for namespace %q — open the lesson page (pharos nav <url>) and retry",
				req.Namespace), 409)
			return
		}

		ch := hub.register(req.Command.ID)
		select {
		case reply := <-ch:
			w.Header().Set("Content-Type", "application/json")
			w.Write(reply)
		case <-time.After(timeout + 2*time.Second): // relay gets a margin over the CLI
			hub.remove(req.Command.ID)
			jsonError(w, fmt.Sprintf("no reply from namespace %q within %s — is the tab still open?", req.Namespace, timeout), 504)
		}
	}
}

// handleWorkbenchReply routes a relay reply to the parked exec call.
// Token-checked: a forged reply is a forged verdict.
func handleWorkbenchReply(hub *execHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkWorkbenchToken(r) {
			jsonError(w, "missing or wrong X-Pharos-Workbench-Token header", 401)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB cap: verdicts are small
		if err != nil {
			jsonError(w, "failed to read request body", 400)
			return
		}
		var msg struct {
			ID    string          `json:"id"`
			Reply json.RawMessage `json:"reply"`
		}
		if err := json.Unmarshal(body, &msg); err != nil || msg.ID == "" {
			jsonError(w, "reply body needs a command id", 400)
			return
		}
		if len(msg.Reply) == 0 {
			// The relay always wraps its reply object under "reply";
			// anything else is a miswired client and fails loud.
			jsonError(w, `reply body needs a "reply" object`, 400)
			return
		}
		if !hub.resolve(msg.ID, msg.Reply) {
			jsonError(w, "no waiting exec call for this command id (expired or unknown)", 404)
			return
		}
		jsonResponse(w, map[string]bool{"routed": true})
	}
}

// ─── Token: load or generate (0600), constant-time check ────────────

var (
	workbenchMu          sync.Mutex
	workbenchTokenValue  string
	workbenchTokenLoaded bool
)

func workbenchToken() string {
	workbenchMu.Lock()
	defer workbenchMu.Unlock()
	if workbenchTokenLoaded {
		return workbenchTokenValue
	}
	workbenchTokenValue = loadOrGenerateWorkbenchToken(config.WorkbenchTokenPath())
	workbenchTokenLoaded = true
	return workbenchTokenValue
}

func loadOrGenerateWorkbenchToken(path string) string {
	if data, err := os.ReadFile(path); err == nil && len(data) >= 32 {
		return string(data)
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		log.Printf("[workbench] token generation failed: %v", err)
		return ""
	}
	token := hex.EncodeToString(buf)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		log.Printf("[workbench] token dir failed: %v", err)
		return ""
	}
	if err := os.WriteFile(path, []byte(token), 0o600); err != nil {
		log.Printf("[workbench] token write failed: %v", err)
		return ""
	}
	return token
}

// setWorkbenchTokenForTest pins the token without touching the real
// config dir, and registers cleanup to restore the lazy load. Tests only.
func setWorkbenchTokenForTest(t *testing.T, token string) {
	workbenchMu.Lock()
	workbenchTokenValue = token
	workbenchTokenLoaded = true
	workbenchMu.Unlock()
	t.Cleanup(func() {
		workbenchMu.Lock()
		workbenchTokenValue = ""
		workbenchTokenLoaded = false
		workbenchMu.Unlock()
	})
}

// checkWorkbenchToken validates the X-Pharos-Workbench-Token header in
// constant time. Generation failure fails closed: no token, no exec.
func checkWorkbenchToken(r *http.Request) bool {
	want := workbenchToken()
	if want == "" {
		return false
	}
	got := r.Header.Get("X-Pharos-Workbench-Token")
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
