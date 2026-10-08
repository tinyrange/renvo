//go:build !renvo

package runtime

import (
	"encoding/binary"
	"testing"
)

func TestNativeMemoryAccessGuards(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Skip(err)
	}
	defer n.Close()
	for _, store := range []bool{false, true} {
		for _, size := range []int{1, 2, 4, 8} {
			var b Builder
			b.Checkpoint(3, 0)
			address := b.Load(0)
			if store {
				b.MemoryStore(address, b.Load(1), size)
			} else {
				b.Store(2, b.MemoryLoad(address, size))
			}
			b.Checkpoint(3, 1)
			ops := b.FinishMemory(3)
			if Validate(ops, 3) == nil {
				t.Fatal("pure evaluator accepted effects")
			}
			entry, err := n.CompileMemory(ops, 3)
			if err != nil {
				t.Fatal(err)
			}
			if err := n.Call(entry, []uint64{4096, 99, 88}); err == nil {
				t.Fatal("memory entry accepted no context")
			}
			if err := n.CallMemory(entry, []uint64{4096, 99, 88}, nil); err == nil {
				t.Fatal("nil context accepted")
			}
			for permissions := uint64(0); permissions <= 7; permissions++ {
				for _, address := range []uint64{4096, 4099, 8192 - uint64(size), 8193 - uint64(size), 8192, 0, 1 << 48, ^uint64(0) - 2} {
					var data [4096]byte
					for i := range data {
						data[i] = byte(i*37 + 129)
					}
					want := data
					clock, pageEpoch := uint64(100), uint64(90)
					ctx := &MemoryContext{Clock: &clock, Retired: 999, Status: 999}
					ctx.Fill(1, &data, permissions, &pageEpoch)
					state := []uint64{address, 0x88776655aabbccdd, 0x1234}
					valid := address >= 4096 && address <= 8192-uint64(size)
					if store {
						valid = valid && permissions&6 == 2
					} else {
						valid = valid && permissions&1 != 0
					}
					expected := state[2]
					if valid {
						offset := int(address - 4096)
						if store {
							for i := 0; i < size; i++ {
								want[offset+i] = byte(state[1] >> uint(i*8))
							}
						} else {
							expected = 0
							for i := 0; i < size; i++ {
								expected |= uint64(data[offset+i]) << uint(i*8)
							}
						}
					}
					if err := n.CallMemory(entry, state, ctx); err != nil {
						t.Fatal(err)
					}
					status, retired := uint64(1), uint64(0)
					if valid {
						status, retired = 0, 1
					}
					if ctx.Status != status || ctx.Retired != retired || state[2] != expected || data != want {
						t.Fatalf("store=%v size=%d permissions=%d address=%x: status=%d retired=%d state=%x expected=%x", store, size, permissions, address, ctx.Status, ctx.Retired, state, expected)
					}
					wantClock, wantEpoch := uint64(100), uint64(90)
					if valid && store {
						wantClock, wantEpoch = 101, 101
					}
					if clock != wantClock || pageEpoch != wantEpoch {
						t.Fatal("bad native store generation", clock, pageEpoch)
					}
					if !valid && ctx.Address != address {
						t.Fatal("wrong slow-path address")
					}
				}
			}
			// Generation exhaustion must never partially write or wrap its clock.
			if store {
				var data [4096]byte
				clock, epoch := ^uint64(0), uint64(3)
				ctx := &MemoryContext{Clock: &clock}
				ctx.Fill(1, &data, 3, &epoch)
				if err := n.CallMemory(entry, []uint64{4096, 99, 88}, ctx); err != nil || ctx.Status != 1 || ctx.Retired != 0 || data[0] != 0 || clock != ^uint64(0) || epoch != 3 {
					t.Fatal("wrapped/partial exhausted store", ctx, err)
				}
			}
		}
	}
}

func TestNativeMemoryCheckpointsAndNoLoadCSE(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Skip(err)
	}
	defer n.Close()
	var b Builder
	b.Checkpoint(4, 0)
	address := b.Load(0)
	first := b.MemoryLoad(address, 8)
	b.Store(1, first)
	b.Checkpoint(4, 1)
	b.MemoryStore(address, b.Constant(42), 8)
	b.Checkpoint(4, 2)
	second := b.MemoryLoad(address, 8)
	b.Store(2, b.Binary(Add, first, second))
	b.Store(3, b.Constant(77))
	b.Checkpoint(4, 3)
	// This load must exit, preserving all preceding state and store effects.
	b.MemoryLoad(b.Constant(8192), 8)
	b.Store(3, b.Constant(88))
	b.Checkpoint(4, 4)
	entry, err := n.CompileMemory(b.FinishMemory(4), 4)
	if err != nil {
		t.Fatal(err)
	}
	var data [4096]byte
	binary.LittleEndian.PutUint64(data[:], 41)
	clock, epoch := uint64(10), uint64(1)
	ctx := &MemoryContext{Clock: &clock}
	ctx.Fill(1, &data, 3, &epoch)
	state := []uint64{4096, 0, 0, 0}
	if err := n.CallMemory(entry, state, ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.Retired != 3 || ctx.Status != 1 || ctx.Address != 8192 || state[1] != 41 || state[2] != 83 || state[3] != 77 || binary.LittleEndian.Uint64(data[:]) != 42 || clock != 11 || epoch != 11 {
		t.Fatal("checkpoint ordering/load forwarding failure", state, ctx.Retired, ctx.Status, clock, epoch)
	}
}

func TestNativeMemoryGuardAndValidation(t *testing.T) {
	n, err := NewNative(16384)
	if err != nil {
		t.Skip(err)
	}
	defer n.Close()
	var b Builder
	b.Store(1, b.Constant(7))
	b.Checkpoint(2, 1)
	b.Guard(b.Load(0))
	b.Store(1, b.Constant(9))
	b.Checkpoint(2, 2)
	entry, err := n.CompileMemory(b.FinishMemory(2), 2)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &MemoryContext{}
	state := []uint64{0, 0}
	if err := n.CallMemory(entry, state, ctx); err != nil || ctx.Status != 2 || ctx.Retired != 1 || state[1] != 7 {
		t.Fatal("failed guard committed following state", state, ctx, err)
	}
	state[0] = 1
	if err := n.CallMemory(entry, state, ctx); err != nil || ctx.Status != 0 || ctx.Retired != 2 || state[1] != 9 {
		t.Fatal("true guard failed", state, ctx, err)
	}
	for _, ops := range [][]Op{
		nil, {{Kind: MemoryLoad, A: 0, Imm: 8}}, {{Kind: Const}, {Kind: MemoryLoad, A: 0, Imm: 3}},
		{{Kind: Const}, {Kind: Progress, Imm: 257}}, {{Kind: Const}, {Kind: MemoryStore, A: 0, B: 0, Imm: 8}, {Kind: Add, A: 1, B: 0}},
		{{Kind: Const}, {Kind: Guard, A: 0}, {Kind: MemoryLoad, A: 1, Imm: 8}},
	} {
		if ValidateMemory(ops, 2) == nil {
			t.Fatal("accepted malformed memory IR", ops)
		}
	}
}
