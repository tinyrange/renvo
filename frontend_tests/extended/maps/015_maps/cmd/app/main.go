package main

func main() {
	m := map[string]int{"a": 16, "b": 4}
	m["a"] = m["a"] + m["b"]
	if m["a"] == 20 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
