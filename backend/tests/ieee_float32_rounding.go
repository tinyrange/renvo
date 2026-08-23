package main

func renvo_runtime_Float32bits(value float32) uint32 { return 0 }

type ieeeFloat32Layout struct {
	first  byte
	value  float32
	second byte
}

func ieeeFloat32Identity(value float32) float32 {
	return value
}

func appMain(args []string) int {
	rounded := float32(16777217)
	if rounded != 16777216 {
		print("FAIL: float32 integer rounding\n")
		return 1
	}

	halfwayEven := float32(0x1.000001p0)
	aboveHalfway := float32(0x1.000003p0)
	if halfwayEven != 1 || aboveHalfway == 1 {
		print("FAIL: float32 round to even\n")
		return 1
	}

	value := ieeeFloat32Identity(float32(0.1))
	if value == 0 || float64(value) == 0.1 {
		print("FAIL: float32 has independent precision\n")
		return 1
	}

	values := [2]float32{1.5, -2.25}
	container := ieeeFloat32Layout{first: 7, value: values[1], second: 9}
	if container.first != 7 || container.value != -2.25 || container.second != 9 {
		print("FAIL: float32 aggregate storage\n")
		return 1
	}

	var boxed interface{} = value
	unboxed, ok := boxed.(float32)
	if !ok || unboxed != value {
		print("FAIL: float32 interface storage\n")
		return 1
	}

	smallest := float32(0x1p-149)
	maxFinite := float32(0x1.fffffep127)
	if renvo_runtime_Float32bits(smallest) != 1 {
		print("FAIL: float32 smallest subnormal\n")
		return 1
	}
	if smallest == 0 {
		print("FAIL: float32 subnormal comparison\n")
		return 1
	}
	if smallest/2 != 0 {
		print("FAIL: float32 underflow\n")
		return 1
	}
	if maxFinite*2 <= maxFinite {
		print("FAIL: float32 overflow\n")
		return 1
	}

	print("PASS\n")
	return 0
}
