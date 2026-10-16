//go:build !renvo

package runtime

import (
	"reflect"
	"renvo.dev/internal/rfenativebridge"
	"testing"
)

// A foreign cycle crosses private register-result loops, an ordinary memory
// leaf, and an ordinary pure leaf. Scalar retirement includes a late memory
// fault and all scheduling tails, after the same edges have already warmed.
func TestNativeMixedFamilyChainPrecisePrefixes(t *testing.T) {
	if !NativeSessionsAvailable() {
		t.Skip("foreign sessions unavailable")
	}
	const words, pc = 12, 11
	for _, profile := range []bool{false, true} {
		for _, preparedPure := range []bool{false, true} {
			n, err := NewNative(1 << 20)
			if err != nil {
				t.Fatal(err)
			}
			defer n.Close()
			n.profileDispatch = profile
			if err = n.PrepareLinks(words, pc); err != nil {
				t.Fatal(err)
			}
			m := new(MemoryContext)
			epoch := uint64(1)
			m.Clock = &epoch
			m.ClaimLinks(n)
			backing := make([]byte, 8192)
			backing[4096] = 23
			m.Fill(1, (*[4096]byte)(backing[4096:]), 1, &epoch)
			entries := [4]int{}
			counts := [4]int{2, 1, 3, 1}
			for phase := 0; phase < 4; phase++ {
				var b Builder
				switch phase {
				case 0:
					b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
				case 2:
					inputs := make([]Value, 6)
					for i := range inputs {
						inputs[i] = b.Load(5 + i)
					}
					b.Store(0, b.Binary(Add, b.Load(0), b.Constant(7)))
					for i, v := range inputs {
						b.Store(5+i, b.Binary(Add, v, b.Constant(uint64(i+1))))
					}
				case 1:
					b.Store(1, b.MemoryLoad(b.Load(2), 8))
				case 3:
					b.Store(3, b.Binary(Add, b.Load(3), b.Constant(3)))
				}
				b.Store(pc, b.Constant(uint64((phase+1)%4*4)))
				if phase == 0 || phase == 2 {
					b.Checkpoint(words, counts[phase])
					b.LoopContinue(b.Constant(0))
					entries[phase], err = n.CompileCodeVersionedLoop(b.FinishMemory(words), words, counts[phase], m)
				} else if phase == 1 {
					b.Checkpoint(words, 1)
					entries[phase], err = n.CompileMemory(b.FinishMemory(words), words)
				} else {
					entries[phase], err = n.Compile(b.Finish(words), words)
				}
				if err != nil {
					t.Fatal(err)
				}
				prefix := [17]uint8{}
				if phase == 1 {
					prefix[1] = 1
				}
				m.PublishLink(uint64(phase*4), entries[phase], counts[phase], prefix)
				if phase != 3 || preparedPure {
					if err = n.PrepareTargetLink(uint64(phase*4), entries[phase], words, counts[phase]); err != nil {
						t.Fatal(err)
					}
				}
			}
			for _, address := range []uint64{4096, 8192, 4096} {
				for _, budget := range []uint64{65536, 1, 2, 3, 4, 5, 6, 7, 8, 13, 63, 64, 65, 4095, 4096, 4097, 65536} {
					for start := 0; start < 4; start++ {
						state := make([]uint64, words)
						state[1], state[2], state[pc] = 999, address, uint64(start*4)
						want := append([]uint64(nil), state...)
						var retired, memories, status uint64
						for {
							phase := int(want[pc] / 4)
							count := uint64(counts[phase])
							if retired+count > budget {
								break
							}
							switch phase {
							case 0:
								want[0]++
							case 2:
								want[0] += 7
								for i := 0; i < 6; i++ {
									want[5+i] += uint64(i + 1)
								}
							case 1:
								if address != 4096 {
									status = 1
								} else {
									want[1] = 23
									memories++
								}
							case 3:
								want[3] += 3
							}
							if status != 0 {
								break
							}
							retired += count
							want[pc] = uint64((phase + 1) % 4 * 4)
						}
						expired, e := n.RunLinkedBackingSessionUntil(state, m, budget, backing, rfenativebridge.MonotonicNS()+1000000000)
						if e != nil || expired || !reflect.DeepEqual(state, want) || m.Total != retired || m.Remaining != budget-retired || m.MemoryTotal != memories || m.Status != status || status != 0 && m.Address != address {
							t.Fatal("mixed scalar prefix", profile, address, budget, start, e, expired, state, want, m.Total, retired, m.MemoryTotal, memories, m.Status, status)
						}
					}
				}
			}
		}

	}
}
