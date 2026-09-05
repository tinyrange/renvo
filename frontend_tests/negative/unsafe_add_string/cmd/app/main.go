package main
import "unsafe"
func main() { _ = unsafe.Add(nil, "bad") }
