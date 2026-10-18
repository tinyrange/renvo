package main

import "fmt"

type result struct {
	value uint64
	index int
}
type callback func(uint64, int) result

var recovered any

func directRecover() { recovered = recover() }
func pad0()          {}
func pad1()          {}
func pad2()          {}
func pad3()          {}
func pad4()          {}
func pad5()          {}
func pad6()          {}
func pad7()          {}
func deferredValue() {
	callbacks := []func(){pad0, pad1, pad2, pad3, pad4, pad5, pad6, pad7, directRecover}
	defer callbacks[len(callbacks)-1]()
	panic("direct-function-value")
}
func nilCall() (okay bool) {
	defer func() { okay = recover() != nil }()
	var f callback
	f(1, 2)
	return false
}

type receiver struct{ offset uint64 }

func (r *receiver) call(value uint64, index int) result {
	r.offset++
	return result{value: value + r.offset, index: index}
}
func f0(value uint64, index int) result {
	return result{value: (value ^ uint64(0)) + uint64(index), index: 0}
}
func f1(value uint64, index int) result {
	return result{value: (value ^ uint64(1)) + uint64(index), index: 1}
}
func f2(value uint64, index int) result {
	return result{value: (value ^ uint64(2)) + uint64(index), index: 2}
}
func f3(value uint64, index int) result {
	return result{value: (value ^ uint64(3)) + uint64(index), index: 3}
}
func f4(value uint64, index int) result {
	return result{value: (value ^ uint64(4)) + uint64(index), index: 4}
}
func f5(value uint64, index int) result {
	return result{value: (value ^ uint64(5)) + uint64(index), index: 5}
}
func f6(value uint64, index int) result {
	return result{value: (value ^ uint64(6)) + uint64(index), index: 6}
}
func f7(value uint64, index int) result {
	return result{value: (value ^ uint64(7)) + uint64(index), index: 7}
}
func f8(value uint64, index int) result {
	return result{value: (value ^ uint64(8)) + uint64(index), index: 8}
}
func f9(value uint64, index int) result {
	return result{value: (value ^ uint64(9)) + uint64(index), index: 9}
}
func f10(value uint64, index int) result {
	return result{value: (value ^ uint64(10)) + uint64(index), index: 10}
}
func f11(value uint64, index int) result {
	return result{value: (value ^ uint64(11)) + uint64(index), index: 11}
}
func f12(value uint64, index int) result {
	return result{value: (value ^ uint64(12)) + uint64(index), index: 12}
}
func f13(value uint64, index int) result {
	return result{value: (value ^ uint64(13)) + uint64(index), index: 13}
}
func f14(value uint64, index int) result {
	return result{value: (value ^ uint64(14)) + uint64(index), index: 14}
}
func f15(value uint64, index int) result {
	return result{value: (value ^ uint64(15)) + uint64(index), index: 15}
}
func f16(value uint64, index int) result {
	return result{value: (value ^ uint64(16)) + uint64(index), index: 16}
}
func f17(value uint64, index int) result {
	return result{value: (value ^ uint64(17)) + uint64(index), index: 17}
}
func f18(value uint64, index int) result {
	return result{value: (value ^ uint64(18)) + uint64(index), index: 18}
}
func f19(value uint64, index int) result {
	return result{value: (value ^ uint64(19)) + uint64(index), index: 19}
}
func f20(value uint64, index int) result {
	return result{value: (value ^ uint64(20)) + uint64(index), index: 20}
}
func f21(value uint64, index int) result {
	return result{value: (value ^ uint64(21)) + uint64(index), index: 21}
}
func f22(value uint64, index int) result {
	return result{value: (value ^ uint64(22)) + uint64(index), index: 22}
}
func f23(value uint64, index int) result {
	return result{value: (value ^ uint64(23)) + uint64(index), index: 23}
}
func f24(value uint64, index int) result {
	return result{value: (value ^ uint64(24)) + uint64(index), index: 24}
}
func f25(value uint64, index int) result {
	return result{value: (value ^ uint64(25)) + uint64(index), index: 25}
}
func f26(value uint64, index int) result {
	return result{value: (value ^ uint64(26)) + uint64(index), index: 26}
}
func f27(value uint64, index int) result {
	return result{value: (value ^ uint64(27)) + uint64(index), index: 27}
}
func f28(value uint64, index int) result {
	return result{value: (value ^ uint64(28)) + uint64(index), index: 28}
}
func f29(value uint64, index int) result {
	return result{value: (value ^ uint64(29)) + uint64(index), index: 29}
}
func f30(value uint64, index int) result {
	return result{value: (value ^ uint64(30)) + uint64(index), index: 30}
}
func f31(value uint64, index int) result {
	return result{value: (value ^ uint64(31)) + uint64(index), index: 31}
}
func main() {
	callbacks := make([]callback, 32)
	callbacks[0] = f0
	callbacks[1] = f1
	callbacks[2] = f2
	callbacks[3] = f3
	callbacks[4] = f4
	callbacks[5] = f5
	callbacks[6] = f6
	callbacks[7] = f7
	callbacks[8] = f8
	callbacks[9] = f9
	callbacks[10] = f10
	callbacks[11] = f11
	callbacks[12] = f12
	callbacks[13] = f13
	callbacks[14] = f14
	callbacks[15] = f15
	callbacks[16] = f16
	callbacks[17] = f17
	callbacks[18] = f18
	callbacks[19] = f19
	callbacks[20] = f20
	callbacks[21] = f21
	callbacks[22] = f22
	callbacks[23] = f23
	callbacks[24] = f24
	callbacks[25] = f25
	callbacks[26] = f26
	callbacks[27] = f27
	callbacks[28] = f28
	callbacks[29] = f29
	callbacks[30] = f30
	callbacks[31] = f31
	for round := 0; round < 3; round++ {
		for i := len(callbacks) - 1; i >= 0; i-- {
			seed := uint64(round+1)<<61 | 0x12345678
			got := callbacks[i](seed, i+round)
			if got.value != (seed^uint64(i))+uint64(i+round) || got.index != i {
				panic("dispatch partition or wide aggregate result")
			}
		}
	}
	r := &receiver{offset: 9}
	callbacks[0] = r.call
	captured := uint64(17)
	callbacks[31] = func(value uint64, index int) result {
		captured += 3
		return result{value: value + captured, index: index + 1}
	}
	if got := callbacks[0](2, 3); got.value != 12 || got.index != 3 || r.offset != 10 {
		panic("bound method environment")
	}
	if got := callbacks[31](2, 3); got.value != 22 || got.index != 4 || captured != 20 {
		panic("closure environment")
	}
	deferredValue()
	if recovered != "direct-function-value" {
		panic("defer-forward recover ownership")
	}
	if !nilCall() {
		panic("nil function value did not panic")
	}
	fmt.Println("PASS")
}
