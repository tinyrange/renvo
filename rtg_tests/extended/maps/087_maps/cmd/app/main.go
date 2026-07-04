package main

func main() {
	m := map[string]int{"a": 3, "b": 11}
	m["a"] = m["a"] + m["b"]
	if m["a"] == 14 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
