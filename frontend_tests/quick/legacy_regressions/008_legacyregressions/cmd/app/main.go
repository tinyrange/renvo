package main

func main() {
	s := []int{1}
	m := map[string]int{"x": s[0]}
	if m["x"] == 1 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
