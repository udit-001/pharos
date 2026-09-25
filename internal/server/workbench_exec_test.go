package server

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

// httptestExec POSTs with the X-Pharos-Workbench-Token header set — the
// exec-channel equivalent of testEnv.post.
func httptestExec(env *testEnv, t *testing.T, target, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Pharos-Workbench-Token", token)
	env.mux.ServeHTTP(rec, req)
	return rec
}

func TestWorkbenchExec_NoLiveTab(t *testing.T) {
	env := newTestEnv(t)
	setWorkbenchTokenForTest(t, "test-token-0123456789abcdef0123456789abcdef")

	rec := httptestExec(env, t, "/api/workspaces/name/alpha/workbench-exec",
		`{"namespace":"lesson-1","command":{"id":"c1","op":"run","sql":"SELECT 1"}}`,
		"test-token-0123456789abcdef0123456789abcdef")
	if rec.Code != 409 {
		t.Fatalf("status = %d, want 409 (no subscriber); body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "no live bench tab") {
		t.Errorf("error should name the fix; body: %s", rec.Body.String())
	}
}

func TestWorkbenchExec_RejectsBadToken(t *testing.T) {
	env := newTestEnv(t)
	setWorkbenchTokenForTest(t, "test-token-0123456789abcdef0123456789abcdef")

	rec := httptestExec(env, t, "/api/workspaces/name/alpha/workbench-exec",
		`{"namespace":"lesson-1","command":{"id":"c1","op":"run","sql":"SELECT 1"}}`,
		"wrong-token")
	if rec.Code != 401 {
		t.Fatalf("status = %d, want 401; body: %s", rec.Code, rec.Body.String())
	}
}

func TestWorkbenchExec_RejectsUnknownOp(t *testing.T) {
	env := newTestEnv(t)
	setWorkbenchTokenForTest(t, "test-token-0123456789abcdef0123456789abcdef")

	rec := httptestExec(env, t, "/api/workspaces/name/alpha/workbench-exec",
		`{"namespace":"lesson-1","command":{"id":"c1","op":"explode"}}`,
		"test-token-0123456789abcdef0123456789abcdef")
	if rec.Code != 400 {
		t.Fatalf("status = %d, want 400; body: %s", rec.Code, rec.Body.String())
	}
}

func TestWorkbenchExec_BroadcastAndReply(t *testing.T) {
	env := newTestEnv(t)
	setWorkbenchTokenForTest(t, "test-token-0123456789abcdef0123456789abcdef")

	// A relay tab: subscribe to the same SSE topic the exec will use.
	ch, unsub := env.broker.Subscribe("workbench:alpha")
	defer unsub()

	// Park the exec call in the background while the "relay" answers.
	type result struct {
		code int
		body string
	}
	done := make(chan result, 1)
	go func() {
		rec := httptestExec(env, t, "/api/workspaces/name/alpha/workbench-exec",
			`{"namespace":"lesson-1","command":{"id":"c1","op":"run","sql":"SELECT title FROM books"}}`,
			"test-token-0123456789abcdef0123456789abcdef")
		done <- result{rec.Code, rec.Body.String()}
	}()

	// The relay receives the broadcast command.
	ev, ok := <-ch
	if !ok || ev.Type != "workbench-command" {
		t.Fatalf("expected workbench-command broadcast; got %+v", ev)
	}
	var cmd map[string]interface{}
	json.Unmarshal(ev.Data, &cmd)
	if cmd["id"] != "c1" || cmd["op"] != "run" || cmd["sql"] != "SELECT title FROM books" {
		t.Fatalf("command payload wrong: %s", string(ev.Data))
	}

	// The relay posts the reply; the exec call must return it verbatim.
	replyBody := `{"id":"c1","reply":{"id":"c1","ok":true,"op":"run","outcome":{"kind":"ok","rowCount":2},"verdict":{"outcome":"pass"}}}`
	rec := httptestExec(env, t, "/api/workspaces/name/alpha/workbench-replies", replyBody,
		"test-token-0123456789abcdef0123456789abcdef")
	if rec.Code != 200 {
		t.Fatalf("reply route status = %d; body: %s", rec.Code, rec.Body.String())
	}

	r := <-done
	if r.code != 200 {
		t.Fatalf("exec status = %d; body: %s", r.code, r.body)
	}
	var reply map[string]interface{}
	json.Unmarshal([]byte(r.body), &reply)
	if reply["ok"] != true || reply["id"] != "c1" {
		t.Errorf("reply not routed verbatim: %s", r.body)
	}
	verdict, _ := reply["verdict"].(map[string]interface{})
	if verdict["outcome"] != "pass" {
		t.Errorf("verdict lost in routing: %s", r.body)
	}
}

func TestWorkbenchReply_UnknownCommandID(t *testing.T) {
	env := newTestEnv(t)
	setWorkbenchTokenForTest(t, "test-token-0123456789abcdef0123456789abcdef")

	rec := httptestExec(env, t, "/api/workspaces/name/alpha/workbench-replies",
		`{"id":"nope","reply":{"ok":true}}`,
		"test-token-0123456789abcdef0123456789abcdef")
	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}
}
