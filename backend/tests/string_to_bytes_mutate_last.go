package main

func appMain(args []string) int {
	bs := []byte("cab")
	for i := 0; i < len(bs); i = i + 1 {
		if i < 2 {
			continue
		}
		bs[i] = 't'
	}
	if bs[2] != 't' {
		print("RENVO-0586 mutate last failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
