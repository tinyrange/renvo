package main

func main() {
	m := map[string]int{"a": 7, "b": 12}
	m["a"] = m["a"] + m["b"]
	if m["a"] == 19 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
