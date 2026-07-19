package main

func main() {
	m := map[string]int{"a": 15, "b": 3}
	m["a"] = m["a"] + m["b"]
	if m["a"] == 18 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
