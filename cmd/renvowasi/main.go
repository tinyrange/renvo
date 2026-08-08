//go:build !renvo_wasi_frontend

package main

import "fmt"

func main() {
	fmt.Println("renvowasi is the size-constrained Renvo-built WASI frontend; see tools/wasm/README.md")
}
