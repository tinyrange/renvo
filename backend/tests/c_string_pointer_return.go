package main

var portableCString [256]int8

func renvo_runtime_CStringPointer(value string) *int8 {
	for i := 0; i < len(value) && i < len(portableCString); i++ {
		portableCString[i] = int8(value[i])
	}
	return &portableCString[0]
}

func cString() *int8 {
	return renvo_runtime_CStringPointer("PASS\n\x00")
}

func appMain(args []string) int {
	value := cString()
	if *value == 80 {
		print("PASS\n")
	}
	return 0
}
