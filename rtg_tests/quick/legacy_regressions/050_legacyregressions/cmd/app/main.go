package main

func main() {
	m := map[string]int{"ab": 7}
	a := "a"
	if m[a+"b"] == 7 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
