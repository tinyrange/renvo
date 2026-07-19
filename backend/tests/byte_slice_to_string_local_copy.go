package main

func makePass() string {
	var out []byte
	out = append(out, 'P')
	out = append(out, 'A')
	out = append(out, 'S')
	out = append(out, 'S')
	out = append(out, '\n')
	return string(out)
}

func appMain(args []string, env []string) int {
	if makePass() != "PASS\n" {
		print("FAIL\n")
		return 1
	}
	print(makePass())
	return 0
}
