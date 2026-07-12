package server

import (
	"log"
	"sync"
)

// Event is the payload broadcast to dashboard subscribers. Type is a
// free-form tag: "changed" (sidebar/grid swap), "page-changed" (content
// area refresh), "navigate" (agent-driven URL navigation). For
// "page-changed", PageType identifies which page type was mutated
// ("lesson", "ref", "glossary", "mission", "notes", "resources",
// "record"); Seq and Slug identify the specific entity when applicable
// (lesson by seq, ref by slug). For "navigate", URL carries the
// dashboard path the browser should navigate to.
type Event struct {
	Type     string `json:"type"`
	PageType string `json:"pageType,omitempty"`
	Seq      int    `json:"seq,omitempty"`
	Slug     string `json:"slug,omitempty"`
	URL      string `json:"url,omitempty"`
}

// brokerBufferSize caps the per-subscriber buffer. A slow client that
// hasn't drained its channel after this many events gets drops rather
// than wedging the broadcast for everyone else.
const brokerBufferSize = 8

// Broker is the in-memory pub/sub for dashboard live-sync events. It is
// a deep module: a small interface (Subscribe + Broadcast) hides all
// fan-out, goroutine safety, and backpressure. In-process (category 1)
// — testable through the interface directly.
//
// Topics are free-form strings ("workspace:alpha", "home"). The broker
// has no workspace semantics; it's a topic router. The scope contract
// (mutations outside the displayed workspace don't disturb it) is a
// property of which topic a page subscribes to, enforced client-side.
type Broker struct {
	mu   sync.Mutex
	subs map[string]map[chan Event]struct{} // topic -> set of subscriber channels
}

// NewBroker constructs an empty Broker.
func NewBroker() *Broker {
	return &Broker{subs: make(map[string]map[chan Event]struct{})}
}

// Subscribe returns a receive-only channel for Events broadcast to the
// given topic, plus an unsubscribe function that removes the channel
// from the broker. The channel is buffered; a slow consumer will have
// events dropped (not block other subscribers). The caller MUST call
// the returned function when done — failing to do so leaks the channel.
//
// The unsubscribe function is safe to call multiple times (idempotent).
func (b *Broker) Subscribe(topic string) (<-chan Event, func()) {
	ch := make(chan Event, brokerBufferSize)
	b.mu.Lock()
	set, ok := b.subs[topic]
	if !ok {
		set = make(map[chan Event]struct{})
		b.subs[topic] = set
	}
	set[ch] = struct{}{}
	b.mu.Unlock()

	var once sync.Once
	unsub := func() {
		once.Do(func() {
			b.mu.Lock()
			delete(set, ch)
			b.mu.Unlock()
		})
	}
	return ch, unsub
}

// Broadcast sends ev to every subscriber of topic. Non-blocking: a
// subscriber whose buffer is full is dropped for this event (logged).
// A missed "changed" just means one stale sidebar until the next
// mutation or manual refresh — preferable to a wedged broker. Returns
// the number of subscribers that received the event.
func (b *Broker) Broadcast(topic string, ev Event) int {
	b.mu.Lock()
	set := b.subs[topic]
	if len(set) == 0 {
		b.mu.Unlock()
		return 0
	}
	// Snapshot under the lock; send without it. A send on a buffered
	// channel won't block, but holding the mutex during send is a bad
	// habit — a future caller reducing the buffer to zero would deadlock.
	chans := make([]chan Event, 0, len(set))
	for ch := range set {
		chans = append(chans, ch)
	}
	b.mu.Unlock()

	delivered := 0
	for _, ch := range chans {
		select {
		case ch <- ev:
			delivered++
		default:
			log.Printf("event broker: dropped %q for slow subscriber", ev.Type)
		}
	}
	return delivered
}

// subscriberCount returns the number of subscribers for a topic. Test
// helper — not part of the external interface surface.
func (b *Broker) subscriberCount(topic string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subs[topic])
}

// BroadcastAll sends ev to every subscriber across all topics. Used by
// "navigate" events that must reach any open tab regardless of which
// workspace it's viewing. Returns the total number of subscribers that
// received the event.
func (b *Broker) BroadcastAll(ev Event) int {
	b.mu.Lock()
	topics := make([]string, 0, len(b.subs))
	for topic, set := range b.subs {
		if len(set) > 0 {
			topics = append(topics, topic)
		}
	}
	b.mu.Unlock()

	delivered := 0
	for _, topic := range topics {
		delivered += b.Broadcast(topic, ev)
	}
	return delivered
}
