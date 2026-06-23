package main

func appMain(args []string) int {
	w16 := 0xffff
	min16 := 0x8000
	w32 := 0xffffffff
	min32 := 0x80000000
	var a int16 = int16(w16)
	if int(a) != -1 {
		print("int16 ff conversion failed\n")
		return 1
	}
	var b int16 = int16(min16)
	if int(b) != -32768 {
		print("int16 sign conversion failed\n")
		return 1
	}
	var c int32 = int32(w32)
	if int(c) != -1 {
		print("int32 ff conversion failed\n")
		return 1
	}
	var d int32 = int32(min32)
	if int(d) != -2147483648 {
		print("int32 sign conversion failed\n")
		return 1
	}
	var e int64 = int64(0x100000000)
	if e != 0x100000000 {
		print("int64 conversion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
