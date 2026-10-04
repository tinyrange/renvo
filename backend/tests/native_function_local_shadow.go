package main

func Unique() int { return 17 }
func Other() int  { return 31 }

func appMain() int {
	var saved func() int = Unique
	var Unique func() int = Other
	if saved() != 17 || Unique() != 31 {
		return 1
	}
	print("PASS\n")
	return 0
}
