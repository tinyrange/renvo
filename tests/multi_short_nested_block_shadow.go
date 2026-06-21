package main

func appMain(args []string) int {
	a := 3
	b := 4
	{
		a, b := 10, 20
		if a+b != 30 {
			print("RTG-1042 inner shadow failed\n")
			return 1
		}
	}
	if a+b != 7 {
		print("RTG-1042 outer values failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
