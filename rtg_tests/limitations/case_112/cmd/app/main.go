package main

import "unsafe"
func main() { var x int; _ = unsafe.Sizeof(x) + unsafe.Sizeof(&x) + unsafe.Sizeof([2]byte{}) }
