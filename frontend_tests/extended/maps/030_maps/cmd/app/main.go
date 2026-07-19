package main

func main() {
	m := map[string]int{"a": 14, "b": 6}
	m["a"] = m["a"] + m["b"]
	if m["a"] == 20 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
