//go:build !renvo

package backendcompiled

import (
	"encoding/binary"
	"testing"
)

// Interpret the straight-line ARM64 entry prefix independently of the host ISA.
// The existing native elapsed tests then exercise the entire body on ARM hosts.
// A contextual time read must leave the live memory-context address available
// for the entry's first status write, not replace it with a retirement count.
func TestARM64ElapsedLoopEntryKeepsMemoryContext(t *testing.T) {
	records := []int{
		1, 0, 0, 1,
		45, 0, 100, 0,
		2, 1, 0, 2,
		0, 0, 0, 0,
		2, 3, 0, 0,
		15, 0, 1, 2,
		0, 0, 0, 1,
		35, 6, 0, 0,
	}
	code, ok := RenvoEmitCodeVersionedLoopBlock(records, 3, 2, true)
	if !ok {
		t.Fatal("elapsed loop emission rejected")
	}
	const frame, context = uint64(0x100000), uint64(0x200000)
	var regs [32]uint64
	regs[1], regs[11], regs[23], regs[29] = context, 0x300000, 16, frame
	memory := map[uint64]uint64{frame - 56: 16, frame - 112: 7, frame - renvoRFESharedPrefix - 16: context, context + 8: 99}
	for at := 0; at+4 <= len(code) && at < 512; at += 4 {
		word := binary.LittleEndian.Uint32(code[at:])
		rd, rn := word&31, word>>5&31
		switch {
		case word&0xff800000 == 0xd2800000: // MOVZ
			regs[rd] = uint64(word>>5&65535) << (16 * (word >> 21 & 3))
		case word&0xffe0ffe0 == 0xaa0003e0: // MOV register (ORR with XZR)
			regs[rd] = regs[word>>16&31]
		case word&0xff000000 == 0x91000000 || word&0xff000000 == 0xd1000000:
			value := uint64(word >> 10 & 4095)
			if word>>22&1 != 0 {
				value <<= 12
			}
			if word>>30&1 != 0 {
				regs[rd] = regs[rn] - value
			} else {
				regs[rd] = regs[rn] + value
			}
		case word&0xffe0fc00 == 0xcb000000 || word&0xffe0fc00 == 0x8b000000:
			value := regs[word>>16&31]
			if word>>30&1 != 0 {
				regs[rd] = regs[rn] - value
			} else {
				regs[rd] = regs[rn] + value
			}
		case word&0xffe00c00 == 0xf8400000 || word&0xffe00c00 == 0xf8000000:
			disp := int64(int32(word<<11) >> 23)
			address := regs[rn] + uint64(disp)
			if word>>22&1 != 0 {
				value, found := memory[address]
				if !found {
					t.Fatalf("uninitialized entry load at %#x", address)
				}
				regs[rd] = value
			} else {
				if address < frame-4096 || address >= frame {
					if address != context+8 || regs[rd] != 0 {
						t.Fatalf("entry corrupted context: store %#x=%#x", address, regs[rd])
					}
					return
				}
				memory[address] = regs[rd]
			}
		case word&0xffc00000 == 0xf9400000 || word&0xffc00000 == 0xf9000000:
			address := regs[rn] + uint64(word>>10&4095)*8
			if word>>22&1 != 0 {
				value, found := memory[address]
				if !found {
					t.Fatalf("uninitialized entry load at %#x", address)
				}
				regs[rd] = value
			} else {
				if address != context+8 || regs[rd] != 0 {
					t.Fatalf("entry corrupted context: store %#x=%#x", address, regs[rd])
				}
				return
			}
		default:
			t.Fatalf("unsupported entry opcode %#08x at %d", word, at)
		}
		regs[31] = 0
	}
	t.Fatal("entry never initialized memory status")
}
