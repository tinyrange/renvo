package main

func frameOperandMutate(value *int) int {
	*value = 9
	return 9
}

func appMain() int {
	padding := [24]int{}
	padding[23] = 17
	bytes := []byte{128, 255}
	b := byte(255)
	if bytes[0] >= b || bytes[1] != b || bytes[1] < b {
		return 1
	}
	s8 := int8(127)
	*s8Address(&s8) = -128
	signed8 := []int8{-128, -1}
	if signed8[0] != s8 || signed8[1] <= s8 {
		return 2
	}
	s16 := int16(32767)
	*s16Address(&s16) = -32768
	signed16 := []int16{-32768, 0}
	if signed16[0] != s16 || signed16[1] <= s16 {
		return 3
	}
	u16 := uint16(65535)
	unsigned16 := []uint16{32768, 65535}
	if unsigned16[0] >= u16 || unsigned16[1] != u16 {
		return 4
	}
	s32 := int32(2147483647)
	*s32Address(&s32) = -2147483647 - 1
	signed32 := []int32{-2147483647 - 1, 0}
	if signed32[0] != s32 || signed32[1] <= s32 {
		return 5
	}
	u32 := uint32(4294967295)
	unsigned32 := []uint32{2147483648, 4294967295}
	if unsigned32[0] >= u32 || unsigned32[1] != u32 {
		return 6
	}
	s64 := int64(-9223372036854775807 - 1)
	u64 := uint64(18446744073709551615)
	if []int64{s64, 0}[1] <= s64 || []uint64{u64, 0}[1] >= u64 {
		return 7
	}
	i, step := 0, 2
	if i+step != len(bytes) || i+step > cap(bytes) || i+step-1 >= len(bytes) {
		return 8
	}
	if i*step+step != len(bytes) || (i+step)*step != 4 {
		return 9
	}
	value := 1
	if frameOperandMutate(&value) != value || value != 9 {
		return 10
	}
	if padding[23] != 17 {
		return 11
	}
	print("PASS\n")
	return 0
}

func s8Address(p *int8) *int8    { return p }
func s16Address(p *int16) *int16 { return p }
func s32Address(p *int32) *int32 { return p }
