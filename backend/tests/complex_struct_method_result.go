package main

type smallComplexBox struct{ Value complex64 }
type largeComplexBox struct{ Value complex128 }

func (b smallComplexBox) Get() complex64  { return b.Value }
func (b largeComplexBox) Get() complex128 { return b.Value }
func smallBox() smallComplexBox           { return smallComplexBox{complex64(1 + 2i)} }
func largeBox() largeComplexBox           { return largeComplexBox{complex128(1 + 2i)} }

func appMain() int {
	b := smallBox()
	if b.Value != complex64(1+2i) {
		return 3
	}
	if b.Get() != complex64(1+2i) {
		return 4
	}
	if smallBox().Get() != complex64(1+2i) {
		return 1
	}
	if largeBox().Get() != complex128(1+2i) {
		return 2
	}
	print("PASS\n")
	return 0
}
