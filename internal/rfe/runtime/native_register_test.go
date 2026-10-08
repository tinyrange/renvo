//go:build !renvo

package runtime

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/bits"
	"strconv"
	"strings"
	"testing"
)

func TestNativeIRRegisterPressureAndStateVersions(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	for _, inputs := range []int{4, 6, 7, 24, 64} {
		var b Builder
		values := make([]Value, inputs)
		for i := range values {
			values[i] = b.Binary(Xor, b.Binary(Mul, b.Load(i), b.Constant(uint64(i*2+3))), b.Constant(0x8000000000000001+uint64(i)))
		}
		// All values live concurrently, and remain live past an old-state overwrite.
		b.Store(0, b.Constant(17))
		sum := b.Constant(0)
		for i := len(values) - 1; i >= 0; i-- {
			sum = b.Binary(Add, sum, values[i])
		}
		b.Store(inputs, sum)
		entry, err := n.Compile(b.Finish(inputs+1), inputs+1)
		if err != nil {
			t.Fatal(err)
		}
		for trial := uint64(0); trial < 32; trial++ {
			state := make([]uint64, inputs+1)
			var want uint64
			for i := 0; i < inputs; i++ {
				state[i] = bits.RotateLeft64(0x87654321fedcba98+trial*7919+uint64(i)*104729, i)
				want += (state[i] * uint64(i*2+3)) ^ (0x8000000000000001 + uint64(i))
			}
			if err = n.Call(entry, state); err != nil {
				t.Fatal(err)
			}
			if state[0] != 17 || state[inputs] != want {
				t.Fatalf("pressure %d trial %d: %x want %x", inputs, trial, state, want)
			}
		}
	}
}
func TestNativeMemoryLiveRegistersAndPreciseSpilledFault(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	for _, fault := range []bool{false, true} {
		var b Builder
		values := make([]Value, 24)
		for i := range values {
			values[i] = b.Binary(Mul, b.Load(i), b.Constant(uint64(i*2+3)))
		}
		b.Checkpoint(26, 0)
		b.MemoryStore(b.Constant(4096), b.Constant(0x88776655aabbccdd), 8)
		b.Store(25, b.Constant(4))
		b.Checkpoint(26, 1)
		loaded := b.MemoryLoad(b.Constant(4096), 4)
		sum := loaded
		for i := len(values) - 1; i >= 0; i-- {
			sum = b.Binary(Xor, sum, values[i])
		}
		b.Store(24, sum)
		b.Store(25, b.Constant(8))
		b.Checkpoint(26, 2)
		address := uint64(4096)
		if fault {
			address = 8192
		}
		b.MemoryLoad(b.Constant(address), 1) // unused result must still fault
		b.Store(25, b.Constant(12))
		b.Checkpoint(26, 3)
		entry, err := n.CompileMemory(b.FinishMemory(26), 26)
		if err != nil {
			t.Fatal(err)
		}
		var page [4096]byte
		clock, epoch := uint64(1), uint64(1)
		m := &MemoryContext{NativeContext: NativeContext{Clock: &clock}}
		m.Fill(1, &page, 3, &epoch)
		state := make([]uint64, 26)
		want := uint64(0xaabbccdd)
		for i := 0; i < 24; i++ {
			state[i] = bits.RotateLeft64(0x123456789abcdef0, i)
			want ^= state[i] * uint64(i*2+3)
		}
		if err = n.CallMemory(entry, state, m); err != nil {
			t.Fatal(err)
		}
		pc, retired, status := uint64(12), uint64(3), uint64(0)
		if fault {
			pc, retired, status = 8, 2, 1
		}
		if state[24] != want || state[25] != pc || m.Retired != retired || m.Status != status || binary.LittleEndian.Uint64(page[:]) != 0x88776655aabbccdd || clock != 2 || epoch != 2 {
			t.Fatal("live register, version or partial-fault mismatch", fault, state, m.Retired, m.Status, clock, epoch)
		}
	}
}
func TestNativeVariableShiftIndependentWords(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	for _, width := range []int{32, 64} {
		for kind := VariableShl; kind <= RotateRight; kind++ {
			var b Builder
			b.Store(2, b.VariableShift(kind, b.Load(0), b.Load(1), width))
			ops := b.Finish(3)
			entry, err := n.Compile(ops, 3)
			if err != nil {
				t.Fatal(err)
			}
			prepared, err := Prepare(ops, 3)
			if err != nil {
				t.Fatal(err)
			}
			for _, value := range []uint64{0, 1, 0x80000000, 0x8000000000000000, 0xdeadbeef76543210, ^uint64(0)} {
				for _, count := range []uint64{0, 1, 7, 15, 31, 32, 33, 63, 64, 65, 127, 128, ^uint64(0)} {
					var want uint64
					shift := count % uint64(width)
					if width == 32 {
						v := uint32(value)
						switch kind {
						case VariableShl:
							want = uint64(v << shift)
						case VariableShr:
							want = uint64(v >> shift)
						case ArithmeticShr:
							want = uint64(uint32(int32(v) >> shift))
						case RotateRight:
							want = uint64(bits.RotateLeft32(v, -int(shift)))
						}
					} else {
						switch kind {
						case VariableShl:
							want = value << shift
						case VariableShr:
							want = value >> shift
						case ArithmeticShr:
							want = uint64(int64(value) >> shift)
						case RotateRight:
							want = bits.RotateLeft64(value, -int(shift))
						}
					}
					state, portable := []uint64{value, count, 17}, []uint64{value, count, 17}
					if err = n.Call(entry, state); err != nil {
						t.Fatal(err)
					}
					var machine IRMachine
					if err = machine.Run(prepared, portable); err != nil {
						t.Fatal(err)
					}
					var folded Builder
					folded.Store(2, folded.VariableShift(kind, folded.Constant(value), folded.Constant(count), width))
					constantState := make([]uint64, 3)
					if err = Interpret(folded.Finish(3), constantState); err != nil {
						t.Fatal(err)
					}
					if state[2] != want || portable[2] != want || constantState[2] != want {
						t.Fatalf("kind%d width%d value%x count%d: native%x portable%x folded%x want%x", kind, width, value, count, state[2], portable[2], constantState[2], want)
					}
				}
			}
		}
	}
	for _, width := range []uint64{0, 1, 16, 33, 128, ^uint64(0)} {
		ops := []Op{{Kind: Const}, {Kind: Const}, {Kind: VariableShl, A: 0, B: 1, Imm: width}}
		if Validate(ops, 1) == nil || ValidateMemory(ops, 1) == nil {
			t.Fatal("invalid shift width accepted", width)
		}
		if _, err := n.Compile(ops, 1); err == nil {
			t.Fatal("native accepted invalid width", width)
		}
	}
}
func TestNativeSymbolRecords(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	var buffer bytes.Buffer
	n.arena.SetSymbolWriter(&buffer)
	var b Builder
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
	entry, err := n.Compile(b.Finish(1), 1)
	if err != nil {
		t.Fatal(err)
	}
	if err = n.NameEntry(entry, "test_leaf"); err != nil {
		t.Fatal(err)
	}
	fields := strings.Fields(buffer.String())
	if len(fields) != 3 || fields[2] != "test_leaf" {
		t.Fatal("bad symbol record", buffer.String())
	}
	address, err := strconv.ParseUint(fields[0], 16, 64)
	if err != nil || address == 0 || address&15 != 0 {
		t.Fatal("bad symbol address", fields[0])
	}
	size, err := strconv.ParseUint(fields[1], 16, 64)
	if err != nil || int(size) != n.Bytes {
		t.Fatal("bad exact symbol length", fields[1], n.Bytes)
	}
	before := buffer.Len()
	for _, name := range []string{"", "bad\nname", "bad\rname", "bad\x00name", strings.Repeat("x", 161)} {
		if n.NameEntry(entry, name) == nil {
			t.Fatal("invalid symbol accepted", fmt.Sprintf("%q", name))
		}
	}
	if n.NameEntry(entry+1, "unaligned") == nil || n.NameEntry(entry+16, "hole") == nil || buffer.Len() != before {
		t.Fatal("invalid entry published")
	}
	if err = n.Close(); err != nil {
		t.Fatal(err)
	}
	// Close disables emission; it cannot expose a new executable address.
	_ = n.NameEntry(entry, "closed")
	if buffer.Len() != before {
		t.Fatal("closed arena published a symbol")
	}
}
