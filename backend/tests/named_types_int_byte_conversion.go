package main

func appMain(args []string) int {
	value := 65
	b := byte(value)
	for i := 0; i < 2; i = i + 1 {
		if i == 0 {
			continue
		}
		if b != 'A' {
			print("RENVO-0661 int byte conversion failed\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
