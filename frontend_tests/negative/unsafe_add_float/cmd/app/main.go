package main
import "unsafe"
func main() { var offset float64 = 1; _ = unsafe.Add(nil, offset) }
