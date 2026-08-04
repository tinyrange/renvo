package main

type nestedDynamicSelectorBuffer struct {
	data  [16]byte
	align uint32
}

func appMain(args []string) int {
	var buffers [2]nestedDynamicSelectorBuffer
	index := 1
	offset := 7
	buffers[index].data[offset] = 41
	if buffers[1].data[7] != 41 || buffers[0].data[7] != 0 {
		print("FAIL: nested dynamic selector assignment\n")
		return 1
	}
	print("PASS\n")
	return 0
}
