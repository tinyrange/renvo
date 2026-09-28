package main

type triple struct{ a, b, c byte }

func appMain() int {
	values := make([]triple, 1, 1)
	values[0] = triple{1, 2, 3}
	original := values
	for i := 1; i < 49; i++ {
		values = append(values, triple{byte(i + 1), byte(i + 2), byte(i + 3)})
	}
	for i := 0; i < len(values); i++ {
		if values[i].a != byte(i+1) || values[i].b != byte(i+2) || values[i].c != byte(i+3) {
			print("FAIL\n")
			return 1
		}
	}
	values[0].a = 99
	if original[0].a != 1 {
		print("FAIL\n")
		return 1
	}
	bytes := make([]byte, 3, 3)
	bytes[0], bytes[1], bytes[2] = 10, 11, 12
	for i := 3; i < 51; i++ {
		bytes = append(bytes, byte(i+10))
	}
	for i := 0; i < len(bytes); i++ {
		if bytes[i] != byte(i+10) {
			print("FAIL\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
