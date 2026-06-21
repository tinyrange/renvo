package main

type rtgMD41Level int

func (level rtgMD41Level) Score(offset int) int {
	return int(level)*10 + offset
}

func appMain(args []string) int {
	level := rtgMD41Level(3)
	if level.Score(7) != 37 {
		print("methods_named_int_receiver failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
