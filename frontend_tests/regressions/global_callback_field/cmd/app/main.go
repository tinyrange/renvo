package main

type Holder struct{ Callback func() }

var calls int
var Callback = func() { calls++ }
var holder = &Holder{}

func replace() { calls += 10 }

func main() {
	holder.Callback = Callback
	holder.Callback()
	Callback()
	Callback = replace
	Callback()
	holder.Callback()
	holder.Callback = Callback
	holder.Callback()
	Callback = nil
	if Callback != nil || calls != 23 {
		panic("global callback assignment")
	}
	println("PASS")
}
