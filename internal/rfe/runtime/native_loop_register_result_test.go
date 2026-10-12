//go:build !renvo

package runtime

import (
	"reflect"
	"renvo.dev/internal/backendcompiled"
	hostruntime "runtime"
	"testing"
)

// Compare architectural operations, precise prefixes and the last published
// result across register-only, genuinely legacy, generic and prepared entries.
// Both calls use the same installed images; the scalar oracle knows no native
// registers, flags, proof offsets or compatibility-wrapper layout.
func TestNativeRegisterLoopResultsAcrossBoundaries(t *testing.T) {
	if hostruntime.GOARCH != "amd64" || !NativeSessionsAvailable() {
		t.Skip("AMD64 foreign boundary unavailable")
	}
	const words, pc = 6, 4
	for _, legacy := range []bool{false, true} {
		n, err := NewNative(1 << 20)
		if err != nil {
			t.Fatal(err)
		}
		if err = n.PrepareLinks(words, pc); err != nil {
			t.Fatal(err)
		}
		var a, b Builder
		a.Store(0, a.Binary(Add, a.Load(0), a.Constant(1)))
		a.Store(pc, a.Constant(4))
		a.Checkpoint(words, 2)
		a.LoopContinue(a.Constant(0))
		first, err := n.CompileLoop(a.FinishMemory(words), words, 2)
		if err != nil {
			t.Fatal(err)
		}
		condition := b.Binary(Less, b.Load(0), b.Load(5))
		b.Store(3, b.Binary(Add, b.Load(3), b.Load(0)))
		b.Store(pc, b.Choose(condition, b.Constant(4), b.Constant(8)))
		b.Checkpoint(words, 1)
		b.RegionGuard(condition)
		b.Store(2, b.MemoryLoad(b.Load(1), 8))
		b.Store(pc, b.Constant(0))
		b.Checkpoint(words, 3)
		b.LoopContinue(b.Constant(0))
		ops := b.FinishMemory(words)
		second := 0
		if legacy {
			code, ok := backendcompiled.RenvoEmitLoopBlock(nativeRecords(ops), words, 3, false)
			if !ok {
				t.Fatal("legacy loop emission")
			}
			second, err = n.arena.InstallLoopBlock(code, words, 3)
		} else {
			second, err = n.CompileLoop(ops, words, 3)
		}
		if err != nil {
			t.Fatal(err)
		}
		for _, prepared := range []bool{false, true} {
			if prepared {
				if err = n.PrepareTargetLink(0, first, words, 2); err != nil {
					t.Fatal(err)
				}
				if err = n.PrepareTargetLink(4, second, words, 3); err != nil {
					t.Fatal(err)
				}
			}
			for _, session := range []bool{false, true} {
				for _, limit := range []uint64{1, 3, 100} {
					for _, readable := range []bool{false, true} {
						for _, budget := range []uint64{1, 2, 3, 4, 5, 6, 7, 9, 10, 11, 17, 63, 64} {
							state := []uint64{0, 4096, 99, 0, 0, limit}
							want := append([]uint64(nil), state...)
							total, memory, iterations, exits, retired, status, address := uint64(0), uint64(0), uint64(0), uint64(0), uint64(0), uint64(0), uint64(0)
							for {
								if want[pc] == 0 {
									if budget-total < 2 {
										break
									}
									want[0]++
									want[pc] = 4
									total += 2
									retired = 2
									iterations++
									exits++
								} else if want[pc] == 4 {
									if budget-total < 3 {
										break
									}
									want[3] += want[0]
									if want[0] >= limit {
										want[pc] = 8
										total++
										retired = 1
										exits++
										break
									}
									if !readable {
										total++
										retired, status, address = 1, 1, 4096
										break
									}
									want[2], want[pc] = 37, 0
									total += 3
									retired = 3
									memory++
									iterations++
									exits++
								} else {
									break
								}
							}
							m := new(MemoryContext)
							page, epoch := new([4096]byte), uint64(1)
							page[0] = 37
							if readable {
								m.Fill(1, page, 1, &epoch)
							}
							m.ClaimLinks(n)
							m.PublishLink(0, first, 2, [17]uint8{})
							m.PublishLink(4, second, 3, [17]uint8{0, 0, 1, 1})
							if session {
								err = n.RunLinkedSession(state, m, budget)
							} else {
								err = n.CallLinked(state, m, budget)
							}
							if err != nil {
								t.Fatal(err)
							}
							if !reflect.DeepEqual(state, want) || m.Total != total || m.Remaining != budget-total || m.MemoryTotal != memory || m.LoopIterations != iterations || m.LoopExits != exits || m.Retired != retired || m.Status != status || m.Address != address || m.CodeView != [4]uint64{} || m.PreparedTargets != 0 || m.DescriptorBase != 0 {
								t.Fatalf("legacy=%v prepared=%v session=%v limit=%d readable=%v budget=%d state=%v want=%v result=(%d,%d,%d,%d,%d,%d,%d) want=(%d,%d,%d,%d,%d,%d,%d)", legacy, prepared, session, limit, readable, budget, state, want, m.Total, m.MemoryTotal, m.LoopIterations, m.LoopExits, m.Retired, m.Status, m.Address, total, memory, iterations, exits, retired, status, address)
							}
						}
					}
				}
			}
		}
		if err = n.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNativeRegisterLoopHintCannotAdmitPrivateBody(t *testing.T) {
	if hostruntime.GOARCH != "amd64" || !NativeSessionsAvailable() {
		t.Skip("AMD64 foreign boundary unavailable")
	}
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(2, 1); err != nil {
		t.Fatal(err)
	}
	entries := [2]int{}
	for i := range entries {
		var b Builder
		b.Store(0, b.Binary(Add, b.Load(0), b.Constant(uint64(i+1))))
		b.Store(1, b.Constant(uint64((1-i)*4)))
		b.Checkpoint(2, 1)
		b.LoopContinue(b.Constant(0))
		entries[i], err = n.CompileLoop(b.FinishMemory(2), 2, 1)
		if err != nil {
			t.Fatal(err)
		}
		if err = n.PrepareTargetLink(uint64(i*4), entries[i], 2, 1); err != nil {
			t.Fatal(err)
		}
	}
	for _, invalid := range []string{"count", "entry", "private-body", "pc", "clear"} {
		m := new(MemoryContext)
		m.ClaimLinks(n)
		m.PublishLink(0, entries[0], 1, [17]uint8{})
		m.PublishLink(4, entries[1], 1, [17]uint8{})
		warm := []uint64{0, 0}
		if err = n.RunLinkedSession(warm, m, 64); err != nil || m.Total != 64 || warm[0] != 96 {
			t.Fatal("warm", err, warm, m.Total)
		}
		switch invalid {
		case "count":
			m.Blocks[1].Instructions = 2
		case "entry":
			m.Blocks[1].Entry++
		case "private-body":
			m.Blocks[1].Entry += 16
		case "pc":
			m.Blocks[1].PC = 4096
		case "clear":
			m.Blocks[1] = NativeLink{}
		}
		state := []uint64{0, 0}
		if err = n.RunLinkedSession(state, m, 64); err != nil {
			t.Fatal(invalid, err)
		}
		if state[0] != 1 || state[1] != 4 || m.Total != 1 || m.Remaining != 63 || m.Retired != 1 || m.Status != 0 {
			t.Fatal("stale hint bypassed descriptor/private-body admission", invalid, state, m.Total, m.Remaining, m.Retired, m.Status)
		}
	}
}
