package main

func smallArrayFill(data []byte, value byte) {
	for i := 0; i < len(data); i++ {
		data[i] = value
	}
}

func appMain() int {
	data := [12]byte{}
	smallArrayFill(data[0:4], 17)
	smallArrayFill(data[4:8], 29)
	smallArrayFill(data[8:12], 43)
	for i := 0; i < 12; i++ {
		want := byte(17)
		if i >= 4 {
			want = 29
		}
		if i >= 8 {
			want = 43
		}
		if data[i] != want {
			print("small array slicing corrupted data\n")
			return 1
		}
	}
	// A pointer-to-array also needs a full slice descriptor, not pointer-sized
	// scratch storage. Exercise both omitted and full slice bounds.
	pointer := &data
	part := pointer[1:3:4]
	if len(part) != 2 || cap(part) != 3 {
		print("pointer array slice bounds\n")
		return 2
	}
	smallArrayFill(part, 55)
	tail := pointer[8:]
	if len(tail) != 4 || tail[0] != 43 || data[0] != 17 || data[1] != 55 || data[3] != 17 {
		print("pointer array slicing corrupted data\n")
		return 3
	}
	print("PASS\n")
	return 0
}
