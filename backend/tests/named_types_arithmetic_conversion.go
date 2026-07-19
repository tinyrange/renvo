package main

func appMain(args []string) int {
	value := 4
	p := &value
	result := int(byte(*p+3)) * 2
	if result != 14 {
		print("RENVO-0665 arithmetic conversion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
