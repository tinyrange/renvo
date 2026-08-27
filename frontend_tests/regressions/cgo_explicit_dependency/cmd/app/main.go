package main

import "example.com/cgodependency/bridge"

func main() {
	if bridge.Value(20) == 42 {
		print("PASS\n")
	}
}
