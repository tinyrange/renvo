package main

type complexComparisonEntry struct {
	key complex128
}

func complexComparisonIdentity(value complex128) complex128 {
	return value
}

func appMain(args []string) int {
	first := complex(float64(-33), float64(0))
	second := complex(float64(-33), float64(1))
	equalFirst := complex(real(first), imag(first))
	returnedSecond := complexComparisonIdentity(second)
	entries := []complexComparisonEntry{{key: first}, {key: second}}
	if first == second || !(first != second) || equalFirst != first || returnedSecond != second || entries[0].key == entries[1].key || entries[0].key != first {
		print("FAIL: complex comparison ignored imaginary part\n")
		return 1
	}
	print("PASS\n")
	return 0
}
