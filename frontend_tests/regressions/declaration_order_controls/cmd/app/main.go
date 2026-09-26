package main

var value = 7

func main() {
	if value != 7 {
		panic("package before local")
	}
	value := 9
	if value != 9 {
		panic("local after declaration")
	}
	if len("abc") != 3 {
		panic("builtin before local")
	}
	len := 4
	if len != 4 {
		panic("local builtin shadow")
	}
	type Node struct {
		next  *Node
		value int
	}
	var n Node
	n.next = &n
	n.value = 11
	if n.next.value != 11 {
		panic("recursive local type")
	}
	goto done
	panic("forward label")
done:
	println("PASS")
}
