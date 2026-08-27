package main

import "C"

func shared(value int) int { return value + 100 }

//export go_callback
func callback(value int) int { return shared(value) }

func main() {
	if shared(2) == 102 && C.shared(2) == 3 && C.call_go(2) == 102 {
		print("PASS\n")
	}
}
