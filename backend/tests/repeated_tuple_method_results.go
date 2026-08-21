package main

type tupleReader struct {
	count int
}

func (reader *tupleReader) next() (line []byte, prefix bool, err error) {
	reader.count++
	if reader.count == 1 {
		return []byte("first"), false, nil
	}
	return []byte("second"), false, nil
}

func appMain() int {
	reader := &tupleReader{}
	line, prefix, err := reader.next()
	if string(line) != "first" || prefix || err != nil {
		print("FAIL\n")
		return 1
	}
	line, prefix, err = reader.next()
	if string(line) == "second" && !prefix && err == nil {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
