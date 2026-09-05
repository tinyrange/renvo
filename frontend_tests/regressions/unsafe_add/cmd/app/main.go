package main

import u "unsafe"

type offset int16

var log int

func pointer(p *byte) u.Pointer { log = log*10 + 1; return u.Pointer(p) }
func distance() uint8           { log = log*10 + 2; return 2 }

func main() {
	data := [8]byte{10, 20, 30, 40, 50, 60, 70, 80}
	base := u.Pointer(&data[0])
	if *(*byte)(u.Add(base, 2)) != 30 {
		panic("positive offset")
	}
	if *(*byte)(u.Add(u.Add(base, 5), offset(-3))) != 30 {
		panic("negative named offset")
	}
	if *(*byte)(u.Add(base, uint64(7))) != 80 {
		panic("unsigned offset")
	}
	*(*byte)(u.Add(base, uintptr(3))) = 99
	if data[3] != 99 {
		panic("write through added pointer")
	}
	if *(*byte)(u.Add(pointer(&data[0]), distance())) != 30 || log != 12 {
		panic("evaluation order")
	}
	if u.Add(base, 0) != base || u.Add(nil, 0) != nil {
		panic("zero offset")
	}
	if *(*byte)(u.Add(base, 2.0)) != 30 || *(*byte)(u.Add(base, 20e-1)) != 30 {
		panic("integral untyped constants")
	}
	if *(*byte)(dotAdd(base)) != 20 {
		panic("dot import")
	}
	words := [2]uint32{12, 34}
	if *(*uint32)(u.Add(u.Pointer(&words[0]), u.Sizeof(words[0]))) != 34 {
		panic("byte offset for wider element")
	}
	println("PASS")
}
