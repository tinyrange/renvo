package main

type rtgSwitchBox struct {
	code int
	mark byte
}

func appMain(args []string) int {
	box := rtgSwitchBox{}
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
		print("RTG-SWITCH-018 struct assign failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
