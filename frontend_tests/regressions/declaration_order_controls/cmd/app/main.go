package main

var x = 7

func main() {
	if x != 7 {
		panic("package before local")
	}
	x := 2
	if x != 2 {
		panic("local")
	}
	{
		if x != 2 {
			panic("outer before inner")
		}
		x := 3
		if x != 3 {
			panic("inner")
		}
	}
	if len("abc") != 3 {
		panic("builtin before local")
	}
	len := 4
	if len != 4 {
		panic("shadowed builtin")
	}
	goto done
done:
	println("PASS")
}
