package main

func main() {
	m := map[string]int{"a": 1}
	n := m
	n["b"] = 2
	delete(n, "a")
	delete(m, "missing")
	var empty map[string]int
	delete(empty, "missing")
	_, found := m["a"]
	if len(m) == 1 && m["b"] == 2 && !found {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
