//go:build !renvo

package main

import "fmt"

func main() {
	fmt.Println("renvowasilanguageservice is built by tools/wasm/build-browser.sh")
}
