//go:build !renvo_wasi_linker

package main

import "fmt"

func main() { fmt.Println("renvowasilinker is built by tools/wasm/build-browser.sh") }
