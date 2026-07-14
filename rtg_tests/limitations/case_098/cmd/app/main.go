package main

import "unsafe"
type T struct { X int }
func main() { _ = unsafe.Sizeof(T{}) }
