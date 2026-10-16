//go:build !renvo

package runtime

import (
	"encoding/binary"
	"runtime"
	"testing"
)

// A raw private successor obeys the result ABI but deliberately reports invalid
// progress on its second visit. By then the exact same invocation has trained
// and admitted both edges. A warmed private prediction must not skip result
// validation or account the predecessor twice.
func TestNativePrivateSuccessorRetainsProgressValidation(t *testing.T) {
	if runtime.GOARCH != "amd64" || !NativeSessionsAvailable() {
		t.Skip("AMD64 foreign session unavailable")
	}
	const words, pc = 3, 2
	for _, profile := range []bool{false, true} {
		for _, invalid := range []uint32{0, 65537} {
			n, err := NewNative(1 << 20)
			if err != nil {
				t.Fatal(err)
			}
			n.profileDispatch = profile
			if err = n.PrepareLinks(words, pc); err != nil {
				t.Fatal(err)
			}
			m := new(MemoryContext)
			m.ClaimLinks(n)
			var b Builder
			b.Store(1, b.Binary(Add, b.Load(1), b.Constant(10)))
			b.Store(pc, b.Constant(4))
			b.Checkpoint(words, 2)
			b.LoopContinue(b.Constant(0))
			root, err := n.CompileCodeVersionedLoop(b.FinishMemory(words), words, 2, m)
			if err != nil {
				t.Fatal(err)
			}
			code := make([]byte, 16)
			// The public prefix is never executed: this test uses a prepared
			// private target. INC counter; set next PC; branch on first visit.
			code = append(code,
				0x49, 0xff, 0x03,
				0x49, 0xc7, 0x43, 0x10, 0, 0, 0, 0,
				0x49, 0x83, 0x3b, 1,
				0x76, 8,
				0xb8, 0, 0, 0, 0, 0x31, 0xd2, 0xc3,
				0xb8, 1, 0, 0, 0, 0x31, 0xd2, 0xc3)
			binary.LittleEndian.PutUint32(code[34:38], invalid)
			raw, err := n.arena.InstallRegisterLoopBlock(code, words, 1)
			if err != nil {
				t.Fatal(err)
			}
			for _, item := range [][3]int{{0, root, 2}, {4, raw, 1}} {
				m.PublishLink(uint64(item[0]), item[1], item[2], [17]uint8{})
				if err = n.PrepareTargetLink(uint64(item[0]), item[1], words, item[2]); err != nil {
					t.Fatal(err)
				}
			}
			for trial := 0; trial < 3; trial++ {
				state := []uint64{0, 0, 0}
				err = n.RunLinkedSession(state, m, 64)
				if err == nil || m.Status != 3 || m.Total != 5 || m.Remaining != 59 || m.MemoryTotal != 0 || state[0] != 2 || state[1] != 20 || state[pc] != 0 {
					t.Fatal("raw result bypassed checked return", profile, invalid, trial, err, state, m.Total, m.Remaining, m.Status)
				}
			}
			n.Close()
		}
	}
}
