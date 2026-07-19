package main

func appMain(args []string) int {
	n, ok, b := 4, true, byte(66)
	if !ok || n+int(b) != 70 {
		print("RENVO-1038 mixed short decl failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
