// Package queryevents provides an in-memory pub/sub bus for live DNS query
// events. The DNS server publishes one Event per query; SSE handlers subscribe
// and receive events without touching the database. The bus also keeps a small
// ring buffer of the most recent events so new subscribers can replay history.
package queryevents

import (
	"sync"
	"time"
)

type Event struct {
	Timestamp time.Time `json:"ts"`
	Domain    string    `json:"domain"`
	ClientIP  string    `json:"client_ip"`
	Blocked   bool      `json:"blocked"`
	Upstream  string    `json:"upstream"`
	LatencyMs int       `json:"latency_ms"`
}

type Bus struct {
	mu       sync.Mutex
	subs     map[chan Event]struct{}
	ring     []Event
	ringHead int
	ringSize int
}

func NewBus(ringSize int) *Bus {
	if ringSize < 1 {
		ringSize = 200
	}
	return &Bus{
		subs:     make(map[chan Event]struct{}),
		ring:     make([]Event, ringSize),
		ringSize: ringSize,
	}
}

func (b *Bus) Publish(e Event) {
	b.mu.Lock()
	b.ring[b.ringHead] = e
	b.ringHead = (b.ringHead + 1) % b.ringSize
	subs := make([]chan Event, 0, len(b.subs))
	for ch := range b.subs {
		subs = append(subs, ch)
	}
	b.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- e:
		default:
		}
	}
}

// Subscribe returns a channel that receives future events and a cleanup
// function the caller MUST invoke when done. The channel is buffered; if the
// consumer falls behind, events are dropped rather than blocking publishers.
func (b *Bus) Subscribe(buffer int) (<-chan Event, func()) {
	if buffer < 1 {
		buffer = 32
	}
	ch := make(chan Event, buffer)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()

	return ch, func() {
		b.mu.Lock()
		delete(b.subs, ch)
		b.mu.Unlock()
		close(ch)
	}
}

// Recent returns up to n most recent events in chronological order (oldest
// first). Useful for replaying history when a new SSE client connects.
func (b *Bus) Recent(n int) []Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	if n > b.ringSize {
		n = b.ringSize
	}

	out := make([]Event, 0, n)
	idx := (b.ringHead - n + b.ringSize) % b.ringSize
	for i := 0; i < n; i++ {
		e := b.ring[idx]
		if !e.Timestamp.IsZero() {
			out = append(out, e)
		}
		idx = (idx + 1) % b.ringSize
	}
	return out
}
