package main

var trace int

func key(step int) string {
	trace = trace*10 + step
	if step == 1 { return "a" }
	if step == 2 { return "b" }
	return "c"
}

func main() {
	m := map[string]int{"a": 0, "b": 2, "c": 3}
	m[key(1)] = m[key(2)] + m[key(3)]
	if trace == 123 && m["a"] == 5 { print("PASS\n"); return }
	print("FAIL\n")
}
