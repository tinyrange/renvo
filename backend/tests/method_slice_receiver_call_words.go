package main

type cwState struct {
	value int
}

func (s *cwState) take(data []byte) {
	s.value = len(data)
}

func appMain() int {
	var state cwState
	state.take([]byte{1, 2, 3})
	if state.value == 3 {
		print("PASS\n")
		return 0
	}
	return 1
}
