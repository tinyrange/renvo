package main

func appMain() int {
	s := "Aé世😀"
	wantIndex := [4]int{0, 1, 3, 6}
	wantRune := [4]int32{65, 233, 19990, 128512}
	count := 0
	for index, value := range s {
		if count >= 4 || index != wantIndex[count] || value != wantRune[count] {
			print("UTF-8 string range failed\n")
			return 1
		}
		count++
	}
	if count != 4 {
		print("UTF-8 string range count failed\n")
		return 1
	}
	invalid := "\xffx"
	count = 0
	for index, value := range invalid {
		if (count == 0 && (index != 0 || value != 65533)) || (count == 1 && (index != 1 || value != 120)) {
			print("invalid UTF-8 string range failed\n")
			return 1
		}
		count++
	}
	if count != 2 {
		print("invalid UTF-8 string range count failed\n")
		return 1
	}
	invalid = "\xc0\x80\xe0\x80\x80\xed\xa0\x80\xf0\x80\x80\x80\xf4\x90\x80\x80\xe2\x82"
	count = 0
	for _, value := range invalid {
		if value != 65533 {
			print("invalid UTF-8 constraint failed\n")
			return 1
		}
		count++
	}
	if count != len(invalid) {
		print("invalid UTF-8 width failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
