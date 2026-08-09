//go:build !renvo

package main

import "os"

func main() {
	os.Exit(run(os.Args[1:]))
}
