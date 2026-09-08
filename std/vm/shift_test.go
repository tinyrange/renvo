package vm

import "testing"

func TestShiftBeyondWordWidth(t *testing.T) {
	for _, count := range []int32{32, 48, 64, 255} {
		for _, c := range []struct {
			op          int
			input, want int32
		}{
			{opShlRegReg, 8, 0}, {opShrRegReg, 8, 0},
			{opShrRegReg, -8, -1}, {opShrUnsignedRegReg, -8, 0},
		} {
			m := machine{}
			m.regs[0], m.regs[1] = c.input, count
			if !m.binary(c.op, 0, 1) || m.regs[0] != c.want {
				t.Fatalf("opcode %d, input %d, count %d: got %d, want %d", c.op, c.input, count, m.regs[0], c.want)
			}
		}
	}
}
