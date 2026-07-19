package main

type Callback func()

type First struct {
	Click Callback
}

type Second struct {
	Click Callback
}

type State struct {
	called int
}

func invoke(value *First) {
	if value.Click != nil {
		value.Click()
	}
}

func main() {
	state := &State{}
	first := First{Click: func() { state.called++ }}
	second := Second{Click: func() { state.called += 2 }}
	invoke(&first)
	second.Click()
	if state.called != 3 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
