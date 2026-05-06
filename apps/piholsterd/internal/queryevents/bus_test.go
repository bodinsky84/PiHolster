package queryevents

import (
	"testing"
	"time"
)

func TestPublishDeliversToSubscriber(t *testing.T) {
	b := NewBus(8)
	ch, cancel := b.Subscribe(4)
	defer cancel()

	want := Event{Timestamp: time.Now(), Domain: "example.com", Blocked: true}
	b.Publish(want)

	select {
	case got := <-ch:
		if got.Domain != want.Domain || got.Blocked != want.Blocked {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber did not receive event")
	}
}

func TestSlowSubscriberDropsRatherThanBlocks(t *testing.T) {
	b := NewBus(8)
	_, cancel := b.Subscribe(1)
	defer cancel()

	for i := 0; i < 100; i++ {
		b.Publish(Event{Timestamp: time.Now(), Domain: "x"})
	}
}

func TestRecentReturnsChronologicalOrder(t *testing.T) {
	b := NewBus(4)
	for _, d := range []string{"a", "b", "c", "d", "e"} {
		b.Publish(Event{Timestamp: time.Now(), Domain: d})
	}

	got := b.Recent(4)
	if len(got) != 4 {
		t.Fatalf("Recent returned %d events, want 4", len(got))
	}
	want := []string{"b", "c", "d", "e"}
	for i, e := range got {
		if e.Domain != want[i] {
			t.Errorf("Recent[%d].Domain = %q, want %q", i, e.Domain, want[i])
		}
	}
}

func TestUnsubscribeRemoves(t *testing.T) {
	b := NewBus(4)
	_, cancel := b.Subscribe(1)
	cancel()

	if got := len(b.subs); got != 0 {
		t.Errorf("after cancel, subs = %d, want 0", got)
	}
}
