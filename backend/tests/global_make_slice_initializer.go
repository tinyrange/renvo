package main

var globalMakeBuffer = make([]byte, 16)

func appMain() int {
	globalMakeBuffer[0] = 'P'
	if len(globalMakeBuffer) != 16 || cap(globalMakeBuffer) != 16 || globalMakeBuffer[0] != 'P' {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
