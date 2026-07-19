package main

func renvo0667TakesByte(b byte) int {
	return int(b)
}

func appMain(args []string) int {
	var xs []int
	xs = append(xs, renvo0667TakesByte(byte(10)))
	if len(xs) != 1 || xs[0] != 10 {
		print("RENVO-0667 argument conversion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
