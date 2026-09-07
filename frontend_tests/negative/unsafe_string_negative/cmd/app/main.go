package main
import "unsafe"
func main() {
	_ = unsafe.String(nil, -1)
}
