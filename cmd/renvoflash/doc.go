//go:build !renvo

// Command renvoflash flashes and hot-reloads supported ESP microcontrollers.
package main

import (
	"os"

	"renvo.dev/internal/espflash"
)

func main() { os.Exit(espflash.Run(os.Args, os.Environ())) }
