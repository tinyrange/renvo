package main

import (
	"os"
	"syscall"
)

func rtgasmAnswer() int

func main() {
	if rtgasmAnswer() != 42 {
		print("RTGASM failure\n")
		return
	}
	if len(os.Args) == 0 || syscall.Getpid() <= 0 || syscall.Access("/unix", syscall.R_OK) != nil {
		print("Unix V7 syscall failure\n")
		return
	}
	print("Hello from Renvo on PDP-11 Unix V7!\n")
}
