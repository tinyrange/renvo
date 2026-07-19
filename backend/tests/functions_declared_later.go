package main

func appMain(args []string) int {
	if renvo0480Later() != 5 {
		print("RENVO-0480 later helper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
func renvo0480Later() int { return 5 }
