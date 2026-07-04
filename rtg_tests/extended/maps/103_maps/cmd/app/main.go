package main

func main() {
	m := map[string]int{"a": 2, "b": 14}
	m["a"] = m["a"] + m["b"]
	if m["a"] == 16 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
