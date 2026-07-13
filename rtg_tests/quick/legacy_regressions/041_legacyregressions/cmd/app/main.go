package main

func main() {
	a := "x"
	b := "xy"[0:1]
	if a == b {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
