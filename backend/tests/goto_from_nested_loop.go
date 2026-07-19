package main

func appMain(args []string) int {
	hit := 0
	for i := 0; i < 3; i = i + 1 {
		for j := 0; j < 3; j = j + 1 {
			if i == 1 && j == 2 {
				hit = i*10 + j
				goto done
			}
		}
	}
done:
	if hit != 12 {
		print("RENVO-0457 nested loop goto failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
