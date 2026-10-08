package main

import emu "renvo.dev/internal/rfe/runtime"

func main() {
	// Renvo-built runners retain the complete native API for portable fallback.
	// A missing loop method used to make importing the AArch64 engine fail.
	if native, err := emu.NewNative(4096); err != nil {
		var unavailable emu.Native
		if _, err := unavailable.CompileLoop(nil, 1, 1); err == nil {
			panic("unavailable native loop accepted")
		}
	} else {
		native.Close()
	}

	page := new([4096]byte)
	epoch := uint64(7)
	memory := emu.MemoryContext{NativeContext: emu.NativeContext{Clock: &epoch, Remaining: 9}}
	memory.Fill(65, page, 3, &epoch)
	if memory.Pages[1].Data != page || memory.Pages[1].Epoch != memory.Clock || memory.Remaining != 9 {
		panic("shared native ABI prefix")
	}
	memory.NativeContext.Total = 4
	memory.Forget(65)
	if memory.Total != 4 || memory.Pages[1].Data != nil {
		panic("shared native ABI mutation")
	}

	var b emu.Builder
	x := b.Load(0)
	r := b.Binary(emu.And, b.Binary(emu.Add, x, b.Constant(1)), b.Constant(65535))
	b.Store(0, r)
	b.Store(1, b.Choose(b.Binary(emu.Equal, r, b.Constant(0)), b.Constant(4), b.Constant(0)))
	b.Store(2, b.Shift(emu.Shr, x, 64))
	state := []uint64{65535, 0, 1}
	if emu.Interpret(b.Finish(3), state) != nil || state[0] != 0 || state[1] != 4 || state[2] != 0 {
		panic("RFE IR semantics")
	}
	var q emu.Queue
	q.Schedule(0, 2, 1, 2)
	q.Schedule(0, 1, 1, 1)
	e, ok := q.Pop(2)
	if !ok || e.Value != 1 {
		panic("RFE event order")
	}
	println("PASS")
}
