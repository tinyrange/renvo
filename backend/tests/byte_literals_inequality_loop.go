package main

func appMain(args []string) int {
	b := byte('O')
	steps := 0
	for b != 'R' {
		b = byte(int(b) + 1)
		steps += 1
	}
	if steps != 3 {
		print("byte_literals_19 loop\n")
		return 1
	}
	print("PASS\n")
	return 0
}
