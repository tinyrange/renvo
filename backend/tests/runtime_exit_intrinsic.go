package main

func renvo_runtime_Exit(code int) {}

func appMain(args []string) int {
	if len(args) == 0 {
		renvo_runtime_Exit(23)
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
