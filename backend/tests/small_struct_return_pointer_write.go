package main

import "unsafe"

type smallStructReturnCarrier struct {
	align byte
	tail  [3]byte
}

func (value *smallStructReturnCarrier) smallStructReturnField() *byte {
	return (*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(value)) + uintptr(3)))
}

func (value *smallStructReturnCarrier) smallStructReturnLabel() *[3]int8 {
	return (*[3]int8)(unsafe.Pointer(uintptr(unsafe.Pointer(value)) + uintptr(0)))
}

func smallStructReturnMake(value byte, label [3]int8) smallStructReturnCarrier {
	var result smallStructReturnCarrier
	*result.smallStructReturnField() = value
	*result.smallStructReturnLabel() = label
	return result
}

func appMain(args []string) int {
	value := smallStructReturnMake(45, [3]int8{104, 105, 0})
	if *value.smallStructReturnField() == 45 &&
		(*value.smallStructReturnLabel())[0] == 104 &&
		(*value.smallStructReturnLabel())[1] == 105 {
		print("PASS\n")
		return 0
	}
	return 1
}
