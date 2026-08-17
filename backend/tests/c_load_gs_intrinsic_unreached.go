package main

func renvo_runtime_CLoadGS(selector uint32) {}

func appMain(args []string) int {
	if len(args) == 99 {
		renvo_runtime_CLoadGS(0)
	}
	print("PASS\n")
	return 0
}
