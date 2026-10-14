package systemclock

import (
	"testing"
	"time"
)

func TestClockFrequencyLongUptimeAndMonotonicity(t *testing.T) {
	for _, freq := range []uint64{100000000, 119318200} {
		elapsed := 200*24*time.Hour + 123456789*time.Nanosecond
		c := WithSource(freq, func() time.Duration { return elapsed }, func(time.Duration) { t.Fatal("unexpected wait") })
		want := uint64(200*24*3600)*freq + uint64(123456789)*freq/1000000000
		if got := c.Read(); got != want {
			t.Fatal(freq, got, want)
		}
		elapsed -= time.Second
		if got := c.Read(); got != want {
			t.Fatal("clock moved backward", got, want)
		}
		elapsed += 2 * time.Second
		if got := c.Read(); got != want+freq {
			t.Fatal("clock failed to catch up", got, want+freq)
		}
	}
}

func TestClockWaitUsesOscillatorAndRoundsUp(t *testing.T) {
	elapsed := time.Duration(0)
	var sleeps []time.Duration
	c := WithSource(119318200, func() time.Duration { return elapsed }, func(d time.Duration) { sleeps = append(sleeps, d); elapsed += d })
	c.WaitUntil(1)
	if c.Read() != 1 || elapsed != 9*time.Nanosecond {
		t.Fatal("early wake or fabricated time", elapsed, c.Read())
	}
	c.WaitUntil(3 * 119318200)
	if c.Read() < 3*119318200 || elapsed < 3*time.Second {
		t.Fatal("long wait", elapsed, c.Read())
	}
	for _, d := range sleeps {
		if d <= 0 || d > time.Second {
			t.Fatal("unbounded wait", d)
		}
	}
	count := len(sleeps)
	c.WaitUntil(0)
	if len(sleeps) != count {
		t.Fatal("past deadline slept")
	}
}

func TestClockTranslationPauseNestedAndResume(t *testing.T) {
	elapsed := 10 * time.Millisecond
	c := WithSource(100000000, func() time.Duration { return elapsed }, func(d time.Duration) { elapsed += d })
	if c.Read() != 1000000 {
		t.Fatal("initial clock")
	}
	c.Pause()
	elapsed += time.Second
	c.Pause()
	elapsed += time.Second
	if c.Read() != 1000000 {
		t.Fatal("translation advanced device oscillator")
	}
	c.Resume()
	elapsed += time.Second
	if c.Read() != 1000000 {
		t.Fatal("nested resume restarted clock too early")
	}
	c.Resume()
	elapsed += time.Millisecond
	if c.Read() != 1100000 {
		t.Fatal("post-translation CPU time lost", c.Read())
	}
	c.WaitUntil(1200000)
	if c.Read() != 1200000 {
		t.Fatal("idle time excluded with translation time")
	}
}
