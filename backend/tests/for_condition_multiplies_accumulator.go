package main

func appMain(args []string) int {
	i := 1
	prod := 1
	for i <= 4 {
		prod = prod * i
		i = i + 1
	}
	if prod != 24 {
		print("RENVO-0382 product loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
