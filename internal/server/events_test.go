package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// ── Broker unit tests ──

func TestBrokerSubscribeAndBroadcast(t *testing.T) {
	b := NewBroker()
	ch, unsub := b.Subscribe("workspace:alpha")
	defer unsub()

	delivered := b.Broadcast("workspace:alpha", Event{Type: "changed"})
	if delivered != 1 {
		t.Fatalf("delivered = %d, want 1", delivered)
	}
	select {
	case ev := <-ch:
		if ev.Type != "changed" {
			t.Fatalf("got type %q, want %q", ev.Type, "changed")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast")
	}
}

func TestBrokerTopicIsolation(t *testing.T) {
	b := NewBroker()
	chAlpha, unsubAlpha := b.Subscribe("workspace:alpha")
	defer unsubAlpha()
	chBeta, unsubBeta := b.Subscribe("workspace:beta")
	defer unsubBeta()

	b.Broadcast("workspace:alpha", Event{Type: "changed", Seq: 1})

	select {
	case ev := <-chAlpha:
		if ev.Seq != 1 {
			t.Fatalf("alpha got seq %d, want 1", ev.Seq)
		}
	case <-time.After(time.Second):
		t.Fatal("alpha should have received the event")
	}

	// Beta must not receive alpha's event.
	select {
	case ev := <-chBeta:
		t.Fatalf("beta should not have received alpha's event, got %+v", ev)
	case <-time.After(100 * time.Millisecond):
		// expected — no event
	}
}

func TestBrokerUnsubscribe(t *testing.T) {
	b := NewBroker()
	_, unsub := b.Subscribe("workspace:alpha")

	if n := b.subscriberCount("workspace:alpha"); n != 1 {
		t.Fatalf("after subscribe, count = %d, want 1", n)
	}

	unsub()

	if n := b.subscriberCount("workspace:alpha"); n != 0 {
		t.Fatalf("after unsubscribe, count = %d, want 0", n)
	}

	// Broadcast to no subscribers — must not block or panic.
	delivered := b.Broadcast("workspace:alpha", Event{Type: "changed"})
	if delivered != 0 {
		t.Fatalf("broadcast to zero subs delivered %d, want 0", delivered)
	}

	// Unsubscribe is idempotent.
	unsub()
}

func TestBrokerDropOnFullBuffer(t *testing.T) {
	b := NewBroker()
	// Subscriber that never drains — buffer fills, subsequent broadcasts drop.
	ch, unsub := b.Subscribe("workspace:alpha")
	defer unsub()

	// Send more than brokerBufferSize events. The first brokerBufferSize
	// land in the buffer; the rest are dropped (non-blocking send).
	for i := 0; i < brokerBufferSize+5; i++ {
		b.Broadcast("workspace:alpha", Event{Type: "changed", Seq: i})
	}

	// Drain the buffer — should get exactly brokerBufferSize events.
	received := 0
drain:
	for {
		select {
		case <-ch:
			received++
		default:
			break drain
		}
	}
	if received != brokerBufferSize {
		t.Fatalf("drained %d events, want %d (buffer size)", received, brokerBufferSize)
	}
}

func TestBrokerConcurrentBroadcast(t *testing.T) {
	b := NewBroker()
	ch, unsub := b.Subscribe("workspace:alpha")
	defer unsub()

	var wg sync.WaitGroup
	const senders = 10
	const perSender = 50
	for i := 0; i < senders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perSender; j++ {
				b.Broadcast("workspace:alpha", Event{Type: "changed"})
			}
		}()
	}
	wg.Wait()

	// Drain — count should be <= senders*perSender (drops allowed) and
	// >= brokerBufferSize (the buffer). No deadlock, no panic.
	received := 0
drain:
	for {
		select {
		case <-ch:
			received++
		default:
			break drain
		}
	}
	if received == 0 {
		t.Fatal("no events received despite concurrent broadcast")
	}
}

// ── Notify handler tests ──

func TestBroadcastAllReachesAllTopics(t *testing.T) {
	b := NewBroker()
	chA, unsubA := b.Subscribe("workspace:alpha")
	defer unsubA()
	chB, unsubB := b.Subscribe("workspace:beta")
	defer unsubB()

	delivered := b.BroadcastAll(Event{Type: "navigate", URL: "/workspace/alpha/lesson/1"})
	if delivered != 2 {
		t.Fatalf("delivered=%d, want 2", delivered)
	}

	for _, ch := range []<-chan Event{chA, chB} {
		select {
		case ev := <-ch:
			if ev.Type != "navigate" || ev.URL != "/workspace/alpha/lesson/1" {
				t.Fatalf("got event %+v, want navigate with URL", ev)
			}
		case <-time.After(time.Second):
			t.Fatal("BroadcastAll did not deliver to subscriber")
		}
	}
}

func TestBroadcastAllZeroSubscribers(t *testing.T) {
	b := NewBroker()
	delivered := b.BroadcastAll(Event{Type: "navigate", URL: "/somewhere"})
	if delivered != 0 {
		t.Fatalf("delivered=%d, want 0 with no subscribers", delivered)
	}
}

func TestNotifyNavigateBroadcastsToAll(t *testing.T) {
	b := NewBroker()
	chA, unsubA := b.Subscribe("workspace:alpha")
	defer unsubA()
	chB, unsubB := b.Subscribe("workspace:beta")
	defer unsubB()

	handler := handleNotify(b)
	body := `{"type":"navigate","url":"/workspace/alpha/lesson/1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/notify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200. body: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Delivered int `json:"delivered"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Delivered != 2 {
		t.Fatalf("delivered=%d, want 2 (both topics)", resp.Delivered)
	}

	// Both subscribers should receive the navigate event with URL.
	for _, ch := range []<-chan Event{chA, chB} {
		select {
		case ev := <-ch:
			if ev.Type != "navigate" || ev.URL != "/workspace/alpha/lesson/1" {
				t.Fatalf("got event %+v, want navigate with URL", ev)
			}
		case <-time.After(time.Second):
			t.Fatal("navigate event not delivered to subscriber")
		}
	}
}

func TestNotifyBroadcastsToBroker(t *testing.T) {
	// Drive the notify handler directly and assert the broker receives
	// the event — no real HTTP streaming needed. This tests the adapter
	// (JSON decode → Broadcast) without the SSE long-poll complexity.
	b := NewBroker()
	ch, unsub := b.Subscribe("workspace:alpha")
	defer unsub()

	handler := handleNotify(b)
	body := `{"topic":"workspace:alpha","type":"page-changed","pageType":"lesson","seq":3}`
	req := httptest.NewRequest(http.MethodPost, "/api/notify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("notify returned %d, want 200. body: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Delivered int `json:"delivered"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response %q: %v", rec.Body.String(), err)
	}
	if resp.Delivered != 1 {
		t.Fatalf("delivered=%d, want 1", resp.Delivered)
	}

	select {
	case ev := <-ch:
		if ev.Type != "page-changed" || ev.PageType != "lesson" || ev.Seq != 3 {
			t.Fatalf("got event %+v, want {page-changed lesson 3}", ev)
		}
		if ev.Slug != "" {
			t.Fatalf("got slug %q, want empty for lesson event", ev.Slug)
		}
	case <-time.After(time.Second):
		t.Fatal("broker did not receive the broadcast from notify handler")
	}
}

func TestNotifyRejectsMissingFields(t *testing.T) {
	b := NewBroker()
	handler := handleNotify(b)

	cases := []struct {
		name string
		body string
	}{
		{"missing type", `{"topic":"workspace:alpha"}`},
		{"missing topic for non-navigate", `{"type":"changed"}`},
		{"empty body", `{}`},
		{"malformed json", `not json`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/notify", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("got %d for %s, want 400", rec.Code, tc.name)
			}
		})
	}
}

func TestNotifyToNoSubscribersIsFast(t *testing.T) {
	b := NewBroker()
	handler := handleNotify(b)

	start := time.Now()
	req := httptest.NewRequest(http.MethodPost, "/api/notify",
		strings.NewReader(`{"topic":"workspace:ghost","type":"changed"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	elapsed := time.Since(start)

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
	var resp struct {
		Delivered int `json:"delivered"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Delivered != 0 {
		t.Fatalf("delivered=%d, want 0 for no subscribers", resp.Delivered)
	}
	// With no subscribers, Broadcast is a map lookup + early return —
	// should be well under 10ms even in CI.
	if elapsed > 50*time.Millisecond {
		t.Fatalf("notify to no subscribers took %v, want < 50ms", elapsed)
	}
}

// ── SSE handler tests ──

func TestSSEMissingTopicReturns400(t *testing.T) {
	b := NewBroker()
	handler := handleSSE(b)

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400 for missing topic", rec.Code)
	}
}

// TestSSEDeliversBroadcast asserts the SSE handler writes a broadcast
// event as a valid SSE data line. Uses a goroutine that broadcasts after
// the handler starts streaming, then cancels the request context to end
// the handler cleanly.
func TestSSEDeliversBroadcast(t *testing.T) {
	b := NewBroker()
	handler := handleSSE(b)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/events?topic=workspace:alpha", nil)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(rec, req)
		close(done)
	}()

	// Give the handler time to subscribe, then broadcast.
	time.Sleep(50 * time.Millisecond)
	b.Broadcast("workspace:alpha", Event{Type: "page-changed", PageType: "lesson", Seq: 7})

	// Let the handler flush, then cancel to end the stream.
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()
	if !strings.Contains(body, ":connected") {
		t.Errorf("SSE response missing :connected comment:\n%s", body)
	}
	if !strings.Contains(body, "data:") {
		t.Fatalf("SSE response missing a data: line:\n%s", body)
	}

	// Parse the data line.
	idx := strings.Index(body, "data:")
	line := body[idx:]
	// Trim to the JSON object — stop at the first \n after data:.
	if nl := strings.IndexByte(line, '\n'); nl != -1 {
		line = line[:nl]
	}
	jsonPart := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	var ev Event
	if err := json.Unmarshal([]byte(jsonPart), &ev); err != nil {
		t.Fatalf("unmarshalling SSE payload %q: %v", jsonPart, err)
	}
	if ev.Type != "page-changed" || ev.PageType != "lesson" || ev.Seq != 7 {
		t.Fatalf("got event %+v, want {page-changed lesson seq=7}", ev)
	}

	// Assert the response headers.
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
}

// TestSSEUnsubscribesOnDisconnect asserts the broker's subscriber count
// drops back to zero after the SSE client disconnects (request context
// cancelled). This is the goroutine-leak guard.
func TestSSEUnsubscribesOnDisconnect(t *testing.T) {
	b := NewBroker()
	handler := handleSSE(b)

	ctx, cancel := context.WithCancel(context.Background())

	req := httptest.NewRequest(http.MethodGet, "/api/events?topic=workspace:alpha", nil)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(rec, req)
		close(done)
	}()

	// Wait for subscription.
	time.Sleep(50 * time.Millisecond)
	if n := b.subscriberCount("workspace:alpha"); n != 1 {
		t.Fatalf("during SSE stream, subscriber count = %d, want 1", n)
	}

	// Disconnect.
	cancel()
	<-done

	if n := b.subscriberCount("workspace:alpha"); n != 0 {
		t.Fatalf("after disconnect, subscriber count = %d, want 0 (leak)", n)
	}
}
