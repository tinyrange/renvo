package main

type nativePackedFields struct {
	A byte
	B uint32
	C uint16
	D byte
}

type nativePackedOuter struct {
	Prefix byte
	Fields nativePackedFields
	Tail   byte
}

type nativePackedPointer struct {
	Prefix byte
	Value  *byte
	Tail   byte
}

// Mark these structures as crossing the native ABI without requiring the
// regression to call a platform-specific symbol.
// rtg:linkstatic unused.dll,consumeNativePackedFields
func consumeNativePackedFields(value *nativePackedFields, outer *nativePackedOuter, pointer *nativePackedPointer) {
}

func appMain(args []string) int {
	value := nativePackedFields{
		C: 0x5566,
		B: 0xd1223344,
		A: 0x7a,
		D: 0x6b,
	}
	if value.A != 0x7a {
		print("FAIL A\n")
		return 1
	}
	if value.B != 0xd1223344 {
		print("FAIL B\n")
		return 1
	}
	if value.C != 0x5566 {
		print("FAIL C\n")
		return 1
	}
	if value.D != 0x6b {
		print("FAIL D\n")
		return 1
	}
	outer := nativePackedOuter{
		Tail:   0x33,
		Fields: value,
		Prefix: 0x22,
	}
	if outer.Prefix != 0x22 || outer.Fields.A != 0x7a || outer.Fields.B != 0xd1223344 || outer.Fields.C != 0x5566 || outer.Fields.D != 0x6b || outer.Tail != 0x33 {
		print("FAIL nested\n")
		return 1
	}
	sentinel := byte(0x44)
	pointer := nativePackedPointer{
		Tail:   0x55,
		Value:  &sentinel,
		Prefix: 0x66,
	}
	if pointer.Prefix != 0x66 || pointer.Value != &sentinel || *pointer.Value != 0x44 || pointer.Tail != 0x55 {
		print("FAIL pointer\n")
		return 1
	}
	print("PASS\n")
	return 0
}
