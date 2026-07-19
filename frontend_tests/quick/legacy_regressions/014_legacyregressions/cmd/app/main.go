package main

func main() {
	x := 1
	inner := 0
	{
		x := 2
		inner = x
	}
	if inner == 2 && x == 1 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
