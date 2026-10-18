package main

type InlineReader interface{ Read() uint64 }
type InlineHolder struct{ Value uint64 }

func (p *InlineHolder) Read() uint64 {
	if p == nil {
		return 17
	}
	return p.Value
}
func inlineTwice(x uint64) uint64 { return x*2 + 1 }
func appMain(args []string) int {
	var p *InlineHolder
	var reader InlineReader = p
	if reader == nil || reader.Read() != 17 {
		print("FAIL typed nil dispatch\n")
		return 1
	}
	var boxed any = p
	q, ok := boxed.(*InlineHolder)
	if !ok || q != nil || boxed == nil {
		print("FAIL typed nil assertion\n")
		return 1
	}
	h := &InlineHolder{Value: 0x1122334455667788}
	reader = h
	boxed = h
	h.Value++
	if reader.Read() != 0x1122334455667789 || boxed.(*InlineHolder) != h {
		print("FAIL live pointer\n")
		return 1
	}
	var fn func(uint64) uint64 = inlineTwice
	boxed = fn
	called, ok := boxed.(func(uint64) uint64)
	if !ok || called == nil || called(0x100000002) != 0x200000005 {
		print("FAIL function assertion\n")
		return 1
	}
	fn = nil
	boxed = fn
	called, ok = boxed.(func(uint64) uint64)
	if !ok || called != nil || boxed == nil {
		print("FAIL typed nil function\n")
		return 1
	}
	boxed = uint64(0xfedcba9876543210)
	if boxed.(uint64) != 0xfedcba9876543210 {
		print("FAIL indirect wide integer\n")
		return 1
	}
	print("PASS\n")
	return 0
}
