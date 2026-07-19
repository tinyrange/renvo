package main

type runtimeFaultRecord struct {
	value int
}

type runtimeFaultHolder struct {
	values  []int
	records []runtimeFaultRecord
	text    string
}

func runtimeFaultZero() int { return 0 }

func runtimeFaultNegativeOne() int { return -1 }

func runtimeFaultRecovered(mode int) (recovered bool) {
	defer func() {
		recovered = recover() != nil
	}()
	values := []int{7}
	array := [1]int{7}
	text := "x"
	index := 2
	switch mode {
	case 0:
		_ = values[index]
	case 1:
		values[index] = 9
	case 2:
		_ = array[index]
	case 3:
		_ = text[index]
	case 4:
		low := -1
		_ = values[low:]
	case 5:
		high := 2
		_ = values[:high]
	case 6:
		low := 1
		high := 0
		_ = values[low:high]
	case 7:
		high := 1
		max := 0
		_ = values[0:high:max]
	case 8:
		max := 2
		_ = values[0:1:max]
	case 9:
		var pointer *runtimeFaultRecord
		_ = pointer.value
	case 10:
		var pointer *int
		_ = *pointer
	case 11:
		_ = 10 / runtimeFaultZero()
	case 12:
		_ = 10 % runtimeFaultZero()
	case 13:
		_ = "x"[index]
	case 14:
		var pointer *int
		*pointer = 1
	case 15:
		var pointer *int
		(*pointer)++
	case 16:
		var pointer *runtimeFaultRecord
		pointer.value = 1
	case 17:
		var pointer *[1]int
		_ = pointer[:]
	case 18:
		var pointer *[1]int
		_ = (*pointer)[0]
	case 19:
		holder := runtimeFaultHolder{records: []runtimeFaultRecord{{value: 1}}}
		_ = holder.records[index].value
	case 20:
		holder := runtimeFaultHolder{records: []runtimeFaultRecord{{value: 1}}}
		holder.records[index].value = 2
	case 21:
		holder := runtimeFaultHolder{values: []int{1}}
		_ = holder.values[index]
	case 22:
		holder := runtimeFaultHolder{text: "x"}
		_ = holder.text[index]
	case 23:
		low := -1
		_ = text[low:]
	case 24:
		high := 2
		_ = text[:high]
	case 25:
		low := 1
		high := 0
		_ = text[low:high]
	}
	return false
}

func appMain() int {
	for mode := 0; mode <= 25; mode++ {
		if !runtimeFaultRecovered(mode) {
			return 1
		}
	}
	minimum := -9223372036854775807
	minimum = minimum - 1
	if minimum/runtimeFaultNegativeOne() != minimum {
		return 2
	}
	if minimum%runtimeFaultNegativeOne() != 0 {
		return 3
	}
	print("PASS\n")
	return 0
}
