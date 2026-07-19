package main

func appMain(args []string) int {
	x := 0
	{

		x = x + 2

		{
			x = x + 3
		}
	}
	if x != 5 {
		print("RENVO-0822 nested block failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
