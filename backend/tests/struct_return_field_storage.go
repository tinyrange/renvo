package main

type fieldStorageResult struct {
	first    byte
	negative int8
	short    int16
	unsigned uint16
	flag     bool
	last     byte
	word     int
	text     string
	values   []int
	pointer  *int
	real     float32
	wide     float64
	complex  complex64
	omitted  [96]byte
}

var fieldStorageCalls int

func fieldStorageValue() int {
	fieldStorageCalls++
	return 73
}

func fieldStorageBuild(values []int, pointer *int) fieldStorageResult {
	return fieldStorageResult{first: 211, negative: -19, short: -1234, unsigned: 54321, flag: true, last: 99,
		word: fieldStorageValue(), text: "fields", values: values, pointer: pointer, real: 1.5, wide: -2.75, complex: complex(3.5, -4.25)}
}

func appMain() int {
	values := make([]int, 1, 2)
	word := 17
	result := fieldStorageBuild(values, &word)
	if result.first != 211 || result.negative != -19 || result.short != -1234 || result.unsigned != 54321 || !result.flag || result.last != 99 {
		panic("narrow field stores")
	}
	if result.word != 73 || fieldStorageCalls != 1 || result.text != "fields" || result.real != 1.5 || result.wide != -2.75 || result.complex != complex(3.5, -4.25) {
		panic("field evaluation")
	}
	result.values[0] = 23
	*result.pointer = 31
	if values[0] != 23 || word != 31 || cap(result.values) != 2 {
		panic("field descriptors")
	}
	for i := 0; i < len(result.omitted); i++ {
		if result.omitted[i] != 0 {
			panic("omitted fields")
		}
	}
	print("PASS\n")
	return 0
}
