package main

type renvoSwitchBox struct {
	code int
	mark byte
}

func appMain(args []string) int {
	box := renvoSwitchBox{}
	key := 2
	switch key {
	case 1:
		box.code = 11
	case 2:
		box.code = 22
		box.mark = 'x'
	default:
		box.code = 33
	}
	if box.code != 22 || box.mark != 'x' {
		print("RENVO-SWITCH-018 struct assign failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
