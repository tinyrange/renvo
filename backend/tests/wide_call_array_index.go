package main

type wideIndexPage struct {
	number      uint64
	data        *byte
	permissions uint64
	epoch       *uint64
}
type wideIndexContext struct{ pages [64]wideIndexPage }

func wideIndexSlot(number uint64) uint64 { return (number ^ (number >> 6) ^ (number >> 12)) & 63 }
func wideIndexFill(context *wideIndexContext, number uint64, data *byte, epoch *uint64) {
	context.pages[wideIndexSlot(number)] = wideIndexPage{number, data, 3, epoch}
}

const wideIndexSets = 4096

func wideIndexLink(pc uint64) uint64 { return ((pc >> 2) ^ (pc >> 12)) & (wideIndexSets - 1) }

var wideIndexCalls int

func wideIndexValue(value uint64) uint64 { wideIndexCalls++; return value }
func wideIndexBounds(value uint64) (caught bool) {
	defer func() { caught = recover() != nil }()
	values := [2]int{11, 22}
	_ = values[wideIndexValue(value)]
	return
}
func appMain() int {
	if wideIndexLink(0xffff800080ca12c0) != 2065 {
		print("FAIL masked high index\n")
		return 4
	}
	base, slot := wideIndexLink(0xffff800080ca12c0), uint64(0)
	var links [4096]uint64
	found := false
	for way := uint64(0); way < 4; way++ {
		i := base + way*4096
		slot, found = i, true
		break
	}
	if !found || slot != 2065 {
		print("FAIL mixed assignment index\n")
		return 6
	}
	links[slot] = 77
	if links[2065] != 77 {
		return 5
	}
	var context wideIndexContext
	var data byte
	epoch := uint64(5)
	for number := uint64(1); number < 16; number++ {
		wideIndexFill(&context, number, &data, &epoch)
		if context.pages[number].number != number || context.pages[number].data != &data || context.pages[number].epoch != &epoch {
			print("FAIL wide call array index\n")
			return 1
		}
	}
	bytes := []byte{11, 22, 33}
	if bytes[wideIndexValue(2)] != 33 || "abc"[wideIndexValue(1)] != 'b' {
		return 2
	}
	if !wideIndexBounds(uint64(1)<<32) || !wideIndexBounds(^uint64(0)) || wideIndexCalls != 4 {
		return 3
	}
	print("PASS\n")
	return 0
}
