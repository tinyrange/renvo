package main

const renvo0493Base = 18

func renvo0493Read() int { return renvo0493Base + 1 }
func appMain(args []string) int {
	if renvo0493Read() != 19 {
		print("RENVO-0493 global const failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
