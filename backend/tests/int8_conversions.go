package main

type int8ConversionRecord struct {
	value int8
}

type int8ConversionArrayRecord struct {
	values [2]int8
}

func int8ConversionByte() byte {
	return byte(254)
}

func int8ConversionRoundTrip(value int8) int8 {
	return value
}

func appMain(args []string) int {
	raw := byte(255)
	converted := int(int8(raw))
	if converted != -1 {
		print("FAIL conversion\n")
		return 1
	}

	dynamic := int8(int8ConversionByte())
	if int(dynamic) != -2 {
		print("FAIL dynamic\n")
		return 1
	}

	raw128 := byte(128)
	values := []int8{int8(127), int8(raw128)}
	if int(values[0]) != 127 || int(values[1]) != -128 {
		print("FAIL slice\n")
		return 1
	}

	raw129 := byte(129)
	record := int8ConversionRecord{value: int8(raw129)}
	if int(record.value) != -127 {
		print("FAIL field\n")
		return 1
	}

	arrayRecord := int8ConversionArrayRecord{values: [2]int8{int8(raw128), int8(raw129)}}
	if int(arrayRecord.values[0]) != -128 || int(arrayRecord.values[1]) != -127 {
		print("FAIL array\n")
		return 1
	}

	assigned := int8(0)
	assigned = int8(raw129)
	if int(int8ConversionRoundTrip(assigned)) != -127 {
		print("FAIL assignment\n")
		return 1
	}

	negative := int8(-120)
	wrapped := negative + int8(-20)
	if int(wrapped) != 116 {
		print("FAIL arithmetic\n")
		return 1
	}
	if !(negative < int8(1)) {
		print("FAIL comparison\n")
		return 1
	}
	if int(negative+1) != -119 || int(1+negative) != -119 || negative+int8(-20) != int8(116) {
		print("FAIL untyped arithmetic\n")
		return 1
	}
	if int(int8(-7)/int8(2)) != -3 || int(int8(-64)>>1) != -32 {
		print("FAIL signed operators\n")
		return 1
	}

	print("PASS\n")
	return 0
}
