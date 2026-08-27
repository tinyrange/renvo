package main

type methods struct {
	allocate func(int32) *byte
	release  func(*byte)
}

type configuration struct {
	methods methods
}

func allocate(size int32) *byte { return nil }
func release(value *byte)       {}

var config = configuration{
	methods: methods{
		allocate: allocate,
		release:  release,
	},
}

func main() {
	if config.methods.allocate == nil || config.methods.release == nil {
		return
	}
	print("PASS\n")
}
