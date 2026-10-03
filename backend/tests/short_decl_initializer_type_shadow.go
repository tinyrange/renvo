package main

type initializerNamed int

func appMain() int {
	int := int(42)
	if int != 42 {
		return 1
	}
	initializerNamed := initializerNamed(7)
	if initializerNamed != 7 {
		return 2
	}
	{
		var string = string("ok")
		if string != "ok" {
			return 3
		}
	}
	print("PASS\n")
	return 0
}
