package main

import "os"
func main() { f, _ := os.OpenFile("x", os.O_RDONLY, 0); _ = f }
