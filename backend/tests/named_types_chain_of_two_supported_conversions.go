package main

type renvo0675Int int
type renvo0675Byte byte

func appMain(args []string) int {
	x := renvo0675Int(90)
	y := renvo0675Byte(int(x))
	z := int(byte(y))
	if z != 90 {
		print("RENVO-0675 conversion chain failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
