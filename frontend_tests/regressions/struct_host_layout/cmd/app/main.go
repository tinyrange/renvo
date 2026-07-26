package main

import (
	"structs"
	"unsafe"
)

type layoutAlias structs.HostLayout

// A package-local name matching the marker implementation must not change
// the identity of structs.HostLayout during unit linking.
type renHL struct {
	Value int
}

type header struct {
	_     layoutAlias
	Kind  byte
	Value uint32
}

type ordinaryInner struct {
	Small byte
	Wide  uint32
}

type hostOuter struct {
	_     structs.HostLayout
	Inner ordinaryInner
	Tail  byte
}

func main() {
	if unsafe.Sizeof(header{}) != 8 || unsafe.Sizeof([2]header{}) != 16 {
		print("header size\n")
		return
	}
	innerSize := unsafe.Sizeof(ordinaryInner{})
	outerSize := unsafe.Sizeof(hostOuter{})
	if innerSize == 8 && outerSize != 12 || innerSize == 16 && outerSize != 24 {
		print("nested layout\n")
		return
	}
	print("PASS\n")
}
