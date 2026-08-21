package time

import (
	"runtime"
	"testing"
)

func TestSleepAndAfter(t *testing.T) {
	start := Now()
	Sleep(20 * Millisecond)
	if elapsed := Since(start); elapsed < 15*Millisecond {
		t.Fatalf("sleep returned early after %v", elapsed)
	}

	start = Now()
	select {
	case when := <-After(20 * Millisecond):
		if when.Sub(start) < 15*Millisecond {
			t.Fatalf("after fired early: %v", when.Sub(start))
		}
	case <-After(2 * Second):
		t.Fatal("after did not fire")
	}
}

func TestTimerStopAndBufferedDelivery(t *testing.T) {
	timer := NewTimer(Hour)
	if !timer.Stop() {
		t.Fatal("stop on fresh timer reported false")
	}
	if timer.Stop() {
		t.Fatal("second stop reported true")
	}
	select {
	case value := <-timer.C:
		t.Fatalf("stopped timer delivered %v", value)
	default:
	}

	fired := NewTimer(5 * Millisecond)
	select {
	case <-fired.C:
	case <-After(2 * Second):
		t.Fatal("short timer did not fire")
	}
	if fired.Stop() {
		t.Fatal("stop after firing reported true")
	}

	// A stopped timer whose task never reads C must not leak a blocked send.
	unused := NewTimer(5 * Millisecond)
	unused.Stop()
	runtime.Gosched()
}
