package main

func main() {
	m := map[string]int{"a": 11, "b": 12}
	m["a"] = m["a"] + m["b"]
	if m["a"] == 23 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
