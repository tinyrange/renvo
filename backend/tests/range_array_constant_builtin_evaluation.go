package main

func rangeArrayNonconstantBuiltin(value complex128) (panicked bool) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	var arrays *[4][2]int
	for range arrays[int(real(value))] {
	}
	return false
}
func rangeArraySliceConversion() (panicked bool) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	var arrays *[4][2]int
	for range arrays[len(string([]byte("x")))] {
	}
	return false
}
func appMain() int {
	var arrays *[4][2]int
	count := 0
	for range arrays[len("x")] {
		count++
	}
	for range arrays[cap([1]int{})] {
		count++
	}
	for range arrays[int(real(complex(1, 0)))] {
		count++
	}
	const local = complex(1, 0)
	for range arrays[int(real(local))] {
		count++
	}
	values := [2]int{1, 2}
	for values := range values {
		count += values
	}
	if count != 9 || !rangeArrayNonconstantBuiltin(complex(1, 0)) || !rangeArraySliceConversion() {
		return 1
	}
	print("PASS\n")
	return 0
}
