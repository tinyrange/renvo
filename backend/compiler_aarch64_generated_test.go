package main

import "testing"

func TestGeneratedAArch64InstructionBindings(t *testing.T) {
	tests := []struct {
		name string
		got  uint32
		want uint32
	}{
		{"ret", rtgAarch64Ret(30), 0xd65f03c0},
		{"add immediate", rtgAarch64AddImmediate(0, 1, 7), 0x91001c20},
		{"move register", rtgAarch64MoveRegister(3, 4), 0xaa0403e3},
		{"movz", rtgAarch64MoveWideZero(0, 0x1234, 16), 0xd2a24680},
		{"movn", rtgAarch64MoveWideNot(9, 0x5678, 32), 0x92cacf09},
		{"movk", rtgAarch64MoveWideKeep(10, 0xabcd, 48), 0xf2f579aa},
		{"add shifted immediate", rtgAarch64AddSubImmediate(0, 1, 3, 1, false), 0x91400c20},
		{"subtract immediate", rtgAarch64AddSubImmediate(2, 2, 9, 0, true), 0xd1002442},
		{"add register", rtgAarch64AddRegister(0, 1, 2), 0x8b020020},
		{"subtract register", rtgAarch64SubtractRegister(3, 4, 5), 0xcb050083},
		{"add shifted register", rtgAarch64AddShiftedRegister(12, 0, 2, 3), 0x8b020c0c},
		{"multiply", rtgAarch64Multiply(0, 1, 2), 0x9b027c20},
		{"load byte unscaled", rtgAarch64LoadUnscaled(0, 1, -1, 1), 0x385ff020},
		{"load word unscaled", rtgAarch64LoadUnscaled(2, 3, 8, 8), 0xf8408062},
		{"load signed half zero", rtgAarch64LoadZeroOffset(4, 5, 2), 0x798000a4},
		{"store byte unscaled", rtgAarch64StoreUnscaled(0, 1, -1, 1), 0x381ff020},
		{"store word unscaled", rtgAarch64StoreUnscaled(2, 3, 8, 8), 0xf8008062},
		{"store half zero", rtgAarch64StoreZeroOffset(4, 5, 2), 0x790000a4},
		{"push", rtgAarch64Push(9), 0xf81f0fe9},
		{"pop", rtgAarch64Pop(9), 0xf84107e9},
		{"compare immediate", rtgAarch64CompareImmediate(3, 17), 0xf100447f},
		{"compare register", rtgAarch64CompareRegister(1, 2), 0xeb02003f},
		{"set condition", rtgAarch64SetCondition(0, 0), 0x9a9f17e0},
		{"shift left", rtgAarch64ShiftLeftImmediate(2, 5), 0xd37be842},
		{"arithmetic shift right", rtgAarch64ShiftRightSignedImmediate(2, 5), 0x9345fc42},
		{"signed divide", rtgAarch64SignedDivide(0, 2, 9), 0x9ac90c40},
		{"multiply subtract", rtgAarch64MultiplySubtract(0, 0, 9, 2), 0x9b098800},
		{"branch", rtgAarch64Branch(8), 0x14000002},
		{"call", rtgAarch64Call(8), 0x94000002},
		{"conditional branch", rtgAarch64ConditionalBranch(8, 1), 0x54000041},
	}
	for _, test := range tests {
		if test.got != test.want {
			t.Errorf("%s = %#08x, want %#08x", test.name, test.got, test.want)
		}
	}
}

func TestRTGEmitterWritesAndPatchesRelativeAddends(t *testing.T) {
	var assembly renvoAsm
	emitter := renvoRTGEmitter(&assembly)
	label := emitter.NewLabel()
	emitter.Rel32Addend(label, 3)
	emitter.Byte(0xaa)
	emitter.Mark(label)
	emitter.Patch()
	if len(assembly.code) != 5 {
		t.Fatalf("code length = %d, want 5", len(assembly.code))
	}
	if got := renvoGet32At(assembly.code, 0); got != 4 {
		t.Fatalf("relative addend = %d, want 4", got)
	}
	if assembly.code[4] != 0xaa {
		t.Fatalf("trailing byte = %#x", assembly.code[4])
	}
}
