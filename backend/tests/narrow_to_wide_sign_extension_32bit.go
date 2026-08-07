package main

type narrowWideInt16 int16

func appMain(args []string) int {
	values := [...]int16{0, -1, -2, 3}
	total := int64(0)
	for index, value := range values {
		total += int64(index) * int64(value)
	}
	if total != 4 {
		print("FAIL: ranged int16 conversion\n")
		return 1
	}
	if int64(int8(-7)) != -7 || int64(int16(-300)) != -300 || int64(int32(-70000)) != -70000 {
		print("FAIL: signed narrow conversion\n")
		return 1
	}
	var named narrowWideInt16 = -1234
	if int64(named) != -1234 {
		print("FAIL: named narrow conversion\n")
		return 1
	}
	print("PASS\n")
	return 0
}
