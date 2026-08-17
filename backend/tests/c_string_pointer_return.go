package main

func renvo_runtime_CStringPointer(value string) *int8 { return nil }

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
