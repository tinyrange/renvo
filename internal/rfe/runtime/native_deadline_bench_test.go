//go:build !renvo

package runtime

import (
	"renvo.dev/internal/rfenativebridge"
	"testing"
)

// Isolate a genuine 64-instruction deadline window from guest code, memory
// translation and boot. Both modes execute the same independently checked
// increment loop and bounded foreign session; only real clock polling differs.
func BenchmarkNativeDeadlineWindow(b *testing.B) {
	if !NativeSessionsAvailable() || rfenativebridge.MonotonicNS() == 0 {
		b.Skip("foreign monotonic clock unavailable")
	}
	for _, timed := range []bool{false, true} {
		name := "untimed"
		if timed {
			name = "monotonic64"
		}
		b.Run(name, func(b *testing.B) {
			n, err := NewNative(1 << 20)
			if err != nil {
				b.Fatal(err)
			}
			defer n.Close()
			if err = n.PrepareLinks(2, 1); err != nil {
				b.Fatal(err)
			}
			backing := make([]byte, 4096)
			epoch := uint64(1)
			m := new(MemoryContext)
			m.Clock = &epoch
			var q Builder
			q.Store(0, q.Binary(Add, q.Load(0), q.Constant(1)))
			q.Store(1, q.Constant(0))
			q.Checkpoint(2, 1)
			q.LoopContinue(q.Constant(1))
			entry, err := n.CompileCodeVersionedLoop(q.FinishMemory(2), 2, 1, m)
			if err != nil {
				b.Fatal(err)
			}
			m.ClaimLinks(n)
			m.PublishLink(0, entry, 1, [17]uint8{})
			if err = n.PrepareTargetLink(0, entry, 2, 1); err != nil {
				b.Fatal(err)
			}
			if !n.AdmitCodeVersionedContext(m) {
				b.Fatal("admission")
			}
			state := []uint64{0, 0}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				state[0] = 0
				if timed {
					hit, e := n.RunLinkedBackingSessionUntil(state, m, 65536, backing, rfenativebridge.MonotonicNS()+1000000000)
					if e != nil || hit {
						b.Fatal(e, hit)
					}
				} else {
					if e := n.RunLinkedBackingSession(state, m, 65536, backing); e != nil {
						b.Fatal(e)
					}
				}
				if state[0] != 65536 || m.Total != 65536 || m.Remaining != 0 || m.Status != 0 {
					b.Fatal("not exact", state, m.Total, m.Remaining, m.Status)
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/1024, "ns/window")
		})
	}
}
