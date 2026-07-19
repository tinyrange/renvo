package main

func appMain(args []string) int {
	bs := []byte("go")
	goto add
add:
	bs = append(bs, '!')
	if len(bs) != 3 || bs[2] != '!' {
		print("RENVO-0587 append converted failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
