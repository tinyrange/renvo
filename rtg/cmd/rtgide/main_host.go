//go:build !rtg

package main

import "os"

func main() { os.Exit(run(os.Args, os.Environ())) }
