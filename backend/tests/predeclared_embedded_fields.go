package main

type PredeclaredEmbeddedInt = struct{ int }
type PredeclaredEmbeddedByte = struct{ byte }
type PredeclaredEmbeddedUint8 = struct{ uint8 }

func predeclaredEmbeddedCopy(v PredeclaredEmbeddedInt) PredeclaredEmbeddedInt { return v }
func appMain() int {
	a := PredeclaredEmbeddedInt{int: 42}
	if predeclaredEmbeddedCopy(a).int != 42 {
		return 1
	}
	b := PredeclaredEmbeddedByte{byte: 43}
	var boxed any = b
	if _, ok := boxed.(PredeclaredEmbeddedUint8); ok {
		return 2
	}
	same, ok := boxed.(PredeclaredEmbeddedByte)
	if !ok || same.byte != 43 {
		return 3
	}
	value := 44
	ptr := struct{ *int }{&value}
	if *ptr.int != 44 {
		return 4
	}
	*ptr.int = 45
	if value != 45 {
		return 5
	}
	print("PASS\n")
	return 0
}
