package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// notifyServer tells the running dashboard that a mutation happened, so
// any open PWA window can live-sync. Best-effort: if no server is
// running, the notify is a silent no-op — the mutation already
// succeeded and the next time the dashboard opens it'll show fresh
// content. Never blocks for more than ~500ms, never returns an error
// that fails the calling command.
//
// topic is the dashboard subscription channel ("workspace:<name>" or
// "home"); typ is the event type ("changed", "page-changed"); seq
// identifies a lesson for "page-changed" with pageType "lesson" (zero
// otherwise).
//
// The CLI never imports the broker — this is the seam between two
// processes. The dashboard happens to be one, but the CLI shouldn't
// care; it just POSTs and moves on.
func notifyServer(topic, typ string, seq int) {
	notifyServerFull(topic, typ, "", seq, "")
}

// notifyPageChanged is the convenience wrapper for content mutations.
// It emits a "page-changed" event to the workspace's subscribers,
// carrying pageType plus seq (for lessons/records) or slug (for refs)
// so the client can match against the currently-open page. Pass zero/""
// for the identifier that doesn't apply.
func notifyPageChanged(wsName, pageType string, seq int, slug string) {
	notifyServerFull("workspace:"+wsName, "page-changed", pageType, seq, slug)
}

func notifyServerFull(topic, typ, pageType string, seq int, slug string) {
	port, ok := runningServerPort()
	if !ok {
		return
	}
	body, _ := json.Marshal(map[string]any{
		"topic":    topic,
		"type":     typ,
		"pageType": pageType,
		"seq":      seq,
		"slug":     slug,
	})
	client := &http.Client{Timeout: 500 * time.Millisecond}
	url := "http://127.0.0.1:" + strconv.Itoa(port) + "/api/notify"
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return
	}
	resp.Body.Close()
}

// runningServerPort reads the pidfile and returns the port of a running
// dashboard server, or (0, false) if none is running / pidfile missing
// / server not actually listening. Reuses the pidfile machinery from
// start.go — the pidfile is the only contract between the CLI and the
// running server process.
func runningServerPort() (int, bool) {
	info, err := readPidFile()
	if err != nil {
		return 0, false
	}
	if !isServerRunning(info.Port) {
		return 0, false
	}
	return info.Port, true
}
