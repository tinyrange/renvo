package main

const stringIndexTen = 2

func appMain(args []string) int {
	s := "red"
	if s[stringIndexTen] != 'd' {
		print("strings_10 const\n")
		return 1
	}
	print("PASS\n")
	return 0
}
