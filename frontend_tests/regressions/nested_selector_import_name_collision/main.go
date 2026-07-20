package main

import "example.com/nested-selector/result"

type builder struct{}

func (b *builder) Result() result.Result {
	return result.Result{Value: 42}
}

type session struct {
	builder *builder
	built   result.Result
}

func main() {
	state := &session{builder: &builder{}}
	state.built = state.builder.Result()
	if state.built.Value != 42 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
