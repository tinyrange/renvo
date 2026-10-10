//go:build !renvo && linux && amd64 && cgo

package linuxuser

import (
	"fmt"
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

// Each owner has a distinct code arena and recovery table. Parallel sessions
// exercise the thread-local fault scope rather than one global current table.
func TestDirectAccessBoundaries(t *testing.T) {
	for _, store := range []bool{false, true} {
		for _, size := range []int{1, 2, 4, 8} {
			t.Run(fmt.Sprintf("store=%v/size=%d", store, size), func(t *testing.T) {
				t.Parallel()
				n, err := emu.NewNative(1 << 20)
				if err != nil {
					t.Fatal(err)
				}
				defer n.Close()
				if err = n.PrepareLinks(3, 2); err != nil {
					t.Fatal(err)
				}
				var b emu.Builder
				a := b.Load(0)
				if store {
					b.MemoryStore(a, b.Constant(0xa1b2c3d4e5f60798), size)
				} else {
					b.Store(1, b.MemoryLoad(a, size))
				}
				b.Checkpoint(3, 1)
				b.LoopContinue(b.Constant(0))
				entry, err := n.CompileLoopMode(b.FinishMemory(3), 3, 1, true)
				if err != nil {
					t.Fatal(err)
				}
				cases := []struct {
					address     uint64
					permissions uint8
					success     bool
				}{
					{4096, 3, true}, {4097, 3, true}, {8192 - uint64(size), 3, true},
					{8192, 3, false}, {^uint64(0) - 3, 3, false}, {AddressLimit, 3, false},
					{DirectWindowSize - 1, 3, false}, {DirectWindowSize + 4096, 3, true},
					{4096, 0, false}, {4096, 4, false},
					{4096, 1, !store}, {4096, 2, false}, {4096, 7, !store},
				}
				if size > 1 {
					cases = append(cases, struct {
						address     uint64
						permissions uint8
						success     bool
					}{8193 - uint64(size), 3, false})
				}
				for _, tc := range cases {
					m, err := NewDirectMemory()
					if err != nil {
						t.Fatal(err)
					}
					if err = m.Map(4096, 4096, 3); err != nil {
						t.Fatal(err)
					}
					if err = m.Map(DirectWindowSize+4096, 4096, 3); err != nil {
						t.Fatal(err)
					}
					// Pre-fill the entire first page: failed cross-page stores must not
					// partially overwrite even the writable part of an invalid access.
					bytes := make([]byte, 4096)
					for i := range bytes {
						bytes[i] = 0x5a
					}
					if err = m.WriteBytes(4096, bytes); err != nil {
						t.Fatal(err)
					}
					if err = m.WriteBytes(DirectWindowSize+4096, bytes); err != nil {
						t.Fatal(err)
					}
					if err = m.Protect(4096, 4096, tc.permissions); err != nil {
						t.Fatal(err)
					}
					ctx := m.NativeContext()
					m.NativeRefill(4096) // Keep the other window on the checked fallback.
					m.NativeRefill(DirectWindowSize + 4096)
					m.NativeRefill(4096)
					m.lookupPage(tc.address / PageSize)
					ctx.ClaimLinks(n)
					ctx.PublishLink(0, entry, 1, [17]uint8{})
					state := []uint64{tc.address, 99, 0}
					if err = n.RunLinkedSession(state, ctx, 64); err != nil {
						t.Fatal(err)
					}
					wantStatus, wantTotal := uint64(1), uint64(0)
					if tc.success {
						wantStatus, wantTotal = 0, 64
					}
					if ctx.Status != wantStatus || ctx.Total != wantTotal || ctx.MemoryTotal != wantTotal || ctx.Remaining != 64-wantTotal {
						t.Fatalf("address=%x perm=%d status=%d total=%d mem=%d remaining=%d", tc.address, tc.permissions, ctx.Status, ctx.Total, ctx.MemoryTotal, ctx.Remaining)
					}
					if !tc.success && ctx.Address != tc.address {
						t.Fatal("fault address", ctx.Address, tc.address)
					}
					if !store {
						want := uint64(99)
						if tc.success {
							want = 0x5a5a5a5a5a5a5a5a
							if size < 8 {
								want &= (uint64(1) << uint(size*8)) - 1
							}
						}
						if state[1] != want {
							t.Fatal("load result", state[1], want)
						}
					} else {
						if err = m.Protect(4096, 4096, 3); err != nil {
							t.Fatal(err)
						}
						if tc.success {
							got, e := m.Read(tc.address, size, false)
							want := uint64(0xa1b2c3d4e5f60798)
							if size < 8 {
								want &= (uint64(1) << uint(size*8)) - 1
							}
							if e != nil || got != want {
								t.Fatal("store result", got, want, e)
							}
						} else {
							got, e := m.ReadBytes(4096, 4096)
							if e != nil {
								t.Fatal(e)
							}
							for _, v := range got {
								if v != 0x5a {
									t.Fatal("failed store partially committed")
								}
							}
						}
					}
					if err = m.Close(); err != nil {
						t.Fatal(err)
					}
				}
			})
		}
	}
}
