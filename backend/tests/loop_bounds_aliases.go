package main

func loopShrinksDescriptor(data []byte) (caught bool) {
	defer func() { caught = recover() != nil }()
	for index := 0; index < len(data); index++ {
		data = data[:0]
		_ = data[index]
	}
	return
}

func loopAliasesDescriptor(data []byte) (caught bool) {
	defer func() { caught = recover() != nil }()
	pointer := &(data)
	for index := 0; index < len(data); index++ {
		*pointer = nil
		_ = data[index]
	}
	return
}

func loopAliasesCounter(data []byte) (caught bool) {
	defer func() { caught = recover() != nil }()
	for index := 0; index < len(data); index++ {
		pointer := &(index)
		*pointer = -1
		_ = data[index]
	}
	return
}

func appMain() int {
	data := []byte{1, 2, 3}
	sum := 0
	for index := 0; index < len(data); index++ {
		sum += int(data[index])
		other := []byte{4, 5}
		for inner := 0; inner < len(other); inner++ {
			sum += int(other[inner])
		}
		sum += int(data[index])
	}
	if sum != 39 || !loopShrinksDescriptor(data) || !loopAliasesDescriptor(data) || !loopAliasesCounter(data) {
		panic("loop bounds proof")
	}
	print("PASS\n")
	return 0
}
