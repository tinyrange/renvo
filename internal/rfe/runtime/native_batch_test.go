//go:build !renvo

package runtime

import (
	"fmt"
	"testing"
	"time"
)

func TestNativeBatchBoundariesAndFailureAccounting(t *testing.T) {
	n, err := NewNative(16384)
	if err != nil {
		t.Skip(err)
	}
	defer n.Close()
	var b Builder
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
	entry, err := n.Compile(b.Finish(1), 1)
	if err != nil {
		t.Fatal(err)
	}
	state := []uint64{0}
	count, err := n.CallBatch(state, 3, func(done int) (int, bool, error) { return entry, done < 3, nil })
	if count != 3 || err != nil || state[0] != 3 {
		t.Fatal("batch count/state", count, state, err)
	}
	count, err = n.CallBatch(state, 3, func(done int) (int, bool, error) { return entry, true, nil })
	if count != 3 || err == nil || state[0] != 6 {
		t.Fatal("call limit bypassed", count, state, err)
	}
	count, err = n.CallBatch(state, 3, func(done int) (int, bool, error) {
		if done == 1 {
			return entry + 1, true, nil
		}
		return entry, true, nil
	})
	if count != 1 || err == nil || state[0] != 7 {
		t.Fatal("invalid entry retired", count, state, err)
	}
	stop := fmt.Errorf("selector fault")
	count, err = n.CallBatch(state, 3, func(done int) (int, bool, error) {
		if done == 1 {
			return 0, false, stop
		}
		return entry, true, nil
	})
	if count != 1 || err != stop || state[0] != 8 {
		t.Fatal("selector fault accounting", count, state, err)
	}
	if count, err := n.CallBatch([]uint64{1, 2}, 3, func(int) (int, bool, error) { return entry, true, nil }); count != 0 || err == nil {
		t.Fatal("batch state shape bypass")
	}
	for _, limit := range []int{0, -1, 257} {
		if _, err := n.CallBatch(state, limit, func(int) (int, bool, error) { t.Fatal("invalid batch called selector"); return entry, true, nil }); err == nil {
			t.Fatal("invalid batch accepted")
		}
	}
	if _, err := n.CallBatch(state, 1, nil); err == nil {
		t.Fatal("nil selector accepted")
	}
	if err := n.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := n.CallBatch(state, 1, func(int) (int, bool, error) { t.Fatal("closed batch called selector"); return entry, true, nil }); err == nil {
		t.Fatal("closed batch accepted")
	}
}

// A selector runs inside arena serialization, not with an unchecked entry
// handle that can outlive Close. The native call must complete before unmap.
func TestNativeBatchCloseSerialization(t *testing.T) {
	n, err := NewNative(16384)
	if err != nil {
		t.Skip(err)
	}
	defer n.Close()
	var b Builder
	b.Store(0, b.Constant(123))
	entry, err := n.Compile(b.Finish(1), 1)
	if err != nil {
		t.Fatal(err)
	}
	entered, release, closeStarted := make(chan struct{}), make(chan struct{}), make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	type result struct {
		count int
		err   error
	}
	finished := make(chan result, 1)
	closed := make(chan error, 1)
	state := []uint64{0}
	go func() {
		count, err := n.CallBatch(state, 1, func(done int) (int, bool, error) {
			if done != 0 {
				return 0, false, nil
			}
			close(entered)
			<-release
			return entry, true, nil
		})
		finished <- result{count, err}
	}()
	<-entered
	go func() { close(closeStarted); closed <- n.Close() }()
	<-closeStarted
	select {
	case err := <-closed:
		t.Fatal("Close returned while batch selector held arena", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	got := <-finished
	if got.count != 1 || got.err != nil || state[0] != 123 {
		t.Fatal("concurrent Close interrupted native call", got, state)
	}
	if err := <-closed; err != nil {
		t.Fatal(err)
	}
}
