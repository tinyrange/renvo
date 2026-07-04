package main

func main() {
	m := map[string]int{"a": 12, "b": 2}
	m["a"] = m["a"] + m["b"]
	if m["a"] == 14 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
