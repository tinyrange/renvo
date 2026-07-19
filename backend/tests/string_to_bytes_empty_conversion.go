package main

func appMain(args []string) int {
	bs := []byte("")
	if len(bs) != 0 {
		print("RENVO-0576 empty conversion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
