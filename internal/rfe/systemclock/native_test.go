//go:build !renvo

package systemclock

import (
	"renvo.dev/internal/rfenativebridge"
	"testing"
)

func TestNativeOscillatorSharesForeignDeadlineDomain(t *testing.T) {
	if !rfenativebridge.Available {
		t.Skip("foreign clock unavailable")
	}
	c := New(1000000)
	if c.nativeStart == 0 {
		t.Skip("native oscillator capability unavailable")
	}
	before := rfenativebridge.MonotonicNS()
	deadline := c.NativeDeadline(c.Read() + 1)
	after := rfenativebridge.MonotonicNS()
	if deadline < before || deadline > after+1000 {
		t.Fatal("clock/deadline domains differ", before, deadline, after)
	}
}
