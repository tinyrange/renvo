package main

type task struct{ Run func() }
type callbackType func()

func main() {
	for _, row := range [][]string{{"one"}, {"two", "three"}} {
		t := &task{}
		n := 0
		t.Run = func() { n += len(row) }
		t.Run()
		t.Run()
		if n != 2*len(row) {
			panic("field closure capture")
		}
	}
	var callback callbackType
	n := 0
	callback = func() { n++ }
	callback()
	if n != 1 {
		panic("local closure capture")
	}
	print("PASS\n")
}
