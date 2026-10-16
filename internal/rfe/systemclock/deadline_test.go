package systemclock

import (
	"testing"
	"time"
)

func TestNativeDeadlineCapabilityAndRounding(t *testing.T) {
	c := WithSource(3, func() time.Duration { return 0 }, func(time.Duration) {})
	if c.NativeDeadline(1) != 0 {
		t.Fatal("custom source acquired native authority")
	}
	c.nativeStart = 17
	if c.NativeDeadline(1) != 333333351 || c.NativeDeadline(3) != 1000000017 {
		t.Fatal("deadline did not round upward")
	}
	if c.NativeDeadline(^uint64(0)) != 0 {
		t.Fatal("deadline overflow")
	}
	c.Pause()
	if c.NativeDeadline(1) != 0 {
		t.Fatal("paused source acquired native authority")
	}
	c.Resume()
	c.withheld = time.Nanosecond
	if c.NativeDeadline(1) != 0 {
		t.Fatal("withheld time acquired native authority")
	}
}
