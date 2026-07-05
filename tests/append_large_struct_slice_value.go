package main

type appendLargeItem struct {
	A int
	B int
	C int
	D int
	E int
	F int
	G int
	H int
	I int
	J int
	K int
}

type appendLargeHolder struct {
	Items []appendLargeItem
}

func appendLargeAdd(h *appendLargeHolder, v int) {
	var it appendLargeItem
	it.A = v
	it.K = v + 1
	items := h.Items
	items = append(items, it)
	h.Items = items
}

func appMain(args []string, env []string) int {
	var h appendLargeHolder
	appendLargeAdd(&h, 11)
	if len(h.Items) != 1 {
		print("FAIL\n")
		return 1
	}
	if h.Items[0].A != 11 {
		print("FAIL\n")
		return 1
	}
	if h.Items[0].K != 12 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
