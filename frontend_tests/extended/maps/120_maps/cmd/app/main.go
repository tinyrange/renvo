package main

func main() {
	m := map[string]int{"a": 2, "b": 5}
	m["a"] = m["a"] + m["b"]
	if m["a"] == 7 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
