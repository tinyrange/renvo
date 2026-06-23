package main

var rtgIntWidths16 []int16
var rtgIntWidths32 []int32

func appMain(args []string) int {
	min16 := 0x8000
	neg32 := 0xffffffff
	rtgIntWidths16 = append(rtgIntWidths16, int16(0x7fff))
	rtgIntWidths16 = append(rtgIntWidths16, int16(min16))
	if len(rtgIntWidths16) != 2 || int(rtgIntWidths16[0]) != 32767 || int(rtgIntWidths16[1]) != -32768 {
		print("int16 append/index failed\n")
		return 1
	}
	rtgIntWidths16[0] += int16(2)
	if int(rtgIntWidths16[0]) != -32767 {
		print("int16 compound slice assignment failed\n")
		return 1
	}
	src := []int32{int32(1), int32(neg32)}
	dst := make([]int32, 2)
	n := copy(dst, src)
	if n != 2 || int(dst[0]) != 1 || int(dst[1]) != -1 {
		print("int32 copy failed\n")
		return 1
	}
	rtgIntWidths32 = append(rtgIntWidths32, src...)
	if len(rtgIntWidths32) != 2 || int(rtgIntWidths32[1]) != -1 {
		print("int32 append expansion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
