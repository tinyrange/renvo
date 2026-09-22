package main
import "unsafe"
func main() {
	_ = unsafe.StringData(1)
}
