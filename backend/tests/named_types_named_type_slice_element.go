package main

type renvo0673Byte byte

func appMain(args []string) int {
	var s []renvo0673Byte
	s = append(s, renvo0673Byte(65))
	s = append(s, renvo0673Byte(66))
	if len(s) != 2 {
		print("RENVO-0673 named slice length failed\n")
		return 1
	}
	if int(s[0])+int(s[1]) != 131 {
		print("RENVO-0673 named slice value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
