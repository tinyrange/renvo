package main

type Address = *int

type Box struct {
	pointer  Address
	callback func() int
}

var object int = 37
var box Box = Box{&object, callback}
var loaded Address = box.pointer
var function func() int = box.callback
var slot *func() int = &function

func callback() int { return 41 }

func appMain(args []string) int {
	if loaded != &object || *loaded != 37 || function() != 41 {
		return 1
	}
	*slot = callback
	if function() != 41 {
		return 2
	}
	local := Box{loaded, function}
	if local.pointer != &object || local.callback() != 41 {
		return 3
	}
	print("PASS\n")
	return 0
}
