package graphics

import "testing"

func TestQueuedEventsReuseBackingStorage(t *testing.T) {
	w := &Window{events: make([]Event, 0, 4)}
	w.queue(Event{Type: EventPointerDown, X: 10})
	w.queue(Event{Type: EventPointerMove, X: 20})
	capacity := cap(w.events)

	event, ok := w.nextQueuedEvent()
	if !ok || event.Type != EventPointerDown || event.X != 10 {
		t.Fatalf("first event = %#v, %v", event, ok)
	}
	if cap(w.events) != capacity {
		t.Fatalf("queue capacity shrank from %d to %d", capacity, cap(w.events))
	}
	event, ok = w.nextQueuedEvent()
	if !ok || event.Type != EventPointerMove || event.X != 20 {
		t.Fatalf("second event = %#v, %v", event, ok)
	}
	if cap(w.events) != capacity || len(w.events) != 0 {
		t.Fatalf("drained queue length/capacity = %d/%d, want 0/%d", len(w.events), cap(w.events), capacity)
	}
}
