package main

type implicitReturnState struct {
	value int
}

func (s *implicitReturnState) finish() {
	s.value = 42
}

func (s *implicitReturnState) set() {
	s.value = 1
	s.finish()
}

func appMain() int {
	state := &implicitReturnState{}
	state.set()
	if state.value == 42 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
