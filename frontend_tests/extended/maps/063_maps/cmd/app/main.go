package main

func main() {
	m := map[string]int{"a": 13, "b": 13}
	m["a"] = m["a"] + m["b"]
	if m["a"] == 26 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
