package main

func main() {
	const count = 2
	type A [count]int
	type B = A
	var a A
	var b B
	var grouped struct{ Values [count]int }
	a[1] = 3
	b[1] = 4
	grouped.Values[1] = 5
	{
		const count = 3
		var inner struct{ Values [count]int }
		inner.Values[2] = 9
		if len(inner.Values) != 3 || inner.Values[2] != 9 {
			panic("shadowed local array type")
		}
	}
	if len(a) != 2 || len(b) != 2 || a[1]+b[1]+grouped.Values[1] != 12 {
		panic("local array type declarations")
	}
	println("PASS")
}
