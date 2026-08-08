//go:build !renvo_wasi_backend

package main

import "fmt"

func main() {
	fmt.Println("renvowasibackend is built by tools/wasm/build.sh")
}
