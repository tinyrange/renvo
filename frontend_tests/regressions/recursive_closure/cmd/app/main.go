package main

type Callback func(int)

func main() {
	total := 0
	var visit Callback
	visit = func(n int) {
		total += n
		if n > 0 {
			visit(n - 1)
		}
	}
	visit(4)
	if total != 10 {
		panic("recursive closure")
	}
	var anonymous func(int)
	anonymous = func(n int) {
		total += n
		if n > 0 { anonymous(n - 1) }
	}
	anonymous(3)
	if total != 16 { panic("anonymous recursive closure") }
	println("PASS")
}
