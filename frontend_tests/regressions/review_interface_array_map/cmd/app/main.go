package main

func main() {
	m := map[interface{}]int{}
	m[[2]int{1, 2}] = 7
	var key interface{} = [2]int{1, 2}
	v, ok := m[key]
	if !ok || v != 7 {
		panic("array key")
	}
	m[key] = 8
	if len(m) != 1 || m[[2]int{1, 2}] != 8 {
		panic("replace")
	}
	println("PASS")
}
