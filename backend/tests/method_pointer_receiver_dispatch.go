package main

type pointerReceiverState struct {
	value int
}

func (state *pointerReceiverState) set(value int) {
	state.value = value
}

func setThroughPointer(state *pointerReceiverState) {
	state.set(42)
}

func appMain() int {
	state := pointerReceiverState{}
	setThroughPointer(&state)
	if state.value != 42 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
