// Package runtime defines the CPU/device boundary for RFE emulators.
// Its access widths, fault categories, and abstract instruction time follow
// tinyrange/renvo_emu's remu-core contract (see docs/rfe.md for provenance).
package runtime

import "fmt"

type Access uint8

const (
	Execute Access = iota
	Read
	Write
)

type FaultKind uint8

const (
	Unmapped FaultKind = iota + 1
	Boundary
	Permission
	Misaligned
	DeviceFault
)

type Fault struct {
	Kind    FaultKind
	Access  Access
	Address uint64
	Width   int
}

func (f *Fault) Error() string {
	return fmt.Sprintf("bus fault %d: access %d at %#x, width %d", f.Kind, f.Access, f.Address, f.Width)
}

// Bus accesses are architectural operations; speculative decoding must never
// call Read on device space. Peek may return false to force ordinary execution.
type Bus interface {
	Load(address uint64, width int, access Access, tick uint64) (uint64, *Fault)
	Store(address uint64, width int, value uint64, tick uint64) *Fault
	Peek(address uint64, width int) (uint64, bool)
}

// Event ordering is (Tick, sequence), including events inserted at equal times.
type Event struct {
	Tick  uint64
	ID    uint64
	Kind  int
	Value uint64
}
type Queue struct {
	events []Event
	next   uint64
}

func (q *Queue) Next() (uint64, bool) {
	if len(q.events) == 0 {
		return 0, false
	}
	return q.events[0].Tick, true
}

func (q *Queue) Schedule(now, at uint64, kind int, value uint64) (uint64, error) {
	if at < now || q.next == ^uint64(0) {
		return 0, fmt.Errorf("invalid event time or exhausted sequence")
	}
	q.next++
	e := Event{at, q.next, kind, value}
	i := len(q.events)
	q.events = append(q.events, e)
	for i > 0 && q.events[i-1].Tick > at {
		q.events[i] = q.events[i-1]
		i--
	}
	q.events[i] = e
	return e.ID, nil
}
func (q *Queue) Pop(now uint64) (Event, bool) {
	if len(q.events) == 0 || q.events[0].Tick > now {
		return Event{}, false
	}
	e := q.events[0]
	q.events = q.events[1:]
	return e, true
}
func (q *Queue) Cancel(id uint64) bool {
	for i, e := range q.events {
		if e.ID == id {
			copy(q.events[i:], q.events[i+1:])
			q.events = q.events[:len(q.events)-1]
			return true
		}
	}
	return false
}
