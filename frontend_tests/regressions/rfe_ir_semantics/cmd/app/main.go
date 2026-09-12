package main

import emu "renvo.dev/internal/rfe/runtime"

func main() {
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
