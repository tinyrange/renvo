package main

func main() {
	m := map[string]int{"a": 17, "b": 4}
	m["a"] = m["a"] + m["b"]
	if m["a"] == 21 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
