package main

func appMain(args []string) int {
	var s []byte
	s = append(s, 'x')
	s = append(s, 'y')
	if len(s) != 2 {
		print("RENVO-0829 slice collection length failed\n")
		return 1
	}
	if s[1] != 'y' {
		print("RENVO-0829 slice collection value failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
