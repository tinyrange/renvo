package clock

import "testing"

type fakeSource struct{ ticks uint32 }

func (s *fakeSource) Ticks() uint32 {
	s.ticks++
	return s.ticks
}
func (s *fakeSource) TicksPerSecond() uint32 { return 1000000 }

func TestDelaysUseMonotonicSource(t *testing.T) {
	source := &fakeSource{}
	clock := New(source)
	clock.DelayMicroseconds(5)
	if source.ticks < 6 {
		t.Fatalf("ticks = %d", source.ticks)
	}
}
