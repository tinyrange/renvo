package main

var renvoIntWidths16 []int16
var renvoIntWidths32 []int32

func appMain(args []string) int {
	min16 := 0x8000
	neg32 := 0xffffffff
	renvoIntWidths16 = append(renvoIntWidths16, int16(0x7fff))
	renvoIntWidths16 = append(renvoIntWidths16, int16(min16))
	if len(renvoIntWidths16) != 2 || int(renvoIntWidths16[0]) != 32767 || int(renvoIntWidths16[1]) != -32768 {
		print("int16 append/index failed\n")
		return 1
	}
	renvoIntWidths16[0] += int16(2)
	if int(renvoIntWidths16[0]) != -32767 {
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
	renvoIntWidths32 = append(renvoIntWidths32, src...)
	if len(renvoIntWidths32) != 2 || int(renvoIntWidths32[1]) != -1 {
		print("int32 append expansion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
