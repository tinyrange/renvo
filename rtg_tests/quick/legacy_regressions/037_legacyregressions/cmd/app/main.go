package main

func main() {
	m := map[string]int{"a": 1}
	x := 2
	m["a"], x = x, m["a"]
	if m["a"] == 2 && x == 1 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
