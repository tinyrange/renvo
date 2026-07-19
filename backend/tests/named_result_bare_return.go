package main

type namedResultRecord struct {
	value int
}

func namedResultFloatCall() float64 {
	return 3.5
}

func namedResultConversion() (result int) {
	result = int(namedResultFloatCall())
	return
}

func namedResultPair() (first int, second int) {
	first = 4
	second = 9
	return
}

func namedResultGroupedPair() (first, second int) {
	first = 12
	second = 13
	return
}

func namedResultArray() (result [2]int) {
	result[0] = 6
	result[1] = 8
	return
}

func namedResultExplicitArray() [2]int {
	return [2]int{10, 11}
}

func namedResultArrayPair() (result [2]int, marker int) {
	result[0] = 14
	result[1] = 15
	marker = 16
	return
}

func namedResultStruct() (result namedResultRecord) {
	result.value = 17
	return
}

func namedResultZeros() (number int, array [2]int, text string, values []int) {
	return
}

func namedResultSlice() (result []int) {
	result = append(result, 7)
	return
}

func namedResultString() (result string) {
	result = "named"
	return
}

func appMain(args []string) int {
	if namedResultConversion() != 3 {
		print("FAIL conversion\n")
		return 1
	}
	first, second := namedResultPair()
	if first != 4 || second != 9 {
		print("FAIL pair\n")
		return 1
	}
	groupedFirst, groupedSecond := namedResultGroupedPair()
	if groupedFirst != 12 || groupedSecond != 13 {
		print("FAIL grouped pair\n")
		return 1
	}
	array := namedResultArray()
	explicitArray := namedResultExplicitArray()
	if array[0] != 6 || array[1] != 8 || explicitArray[0] != 10 || explicitArray[1] != 11 {
		print("FAIL array\n")
		return 1
	}
	pairArray, marker := namedResultArrayPair()
	if pairArray[0] != 14 {
		print("FAIL array pair first\n")
		return 1
	}
	if pairArray[1] != 15 {
		print("FAIL array pair second\n")
		return 1
	}
	if marker != 16 {
		print("FAIL array pair marker\n")
		return 1
	}
	if namedResultStruct().value != 17 {
		print("FAIL struct\n")
		return 1
	}
	zeroNumber, zeroArray, zeroText, zeroValues := namedResultZeros()
	if zeroNumber != 0 || zeroArray[0] != 0 || zeroArray[1] != 0 || zeroText != "" || len(zeroValues) != 0 {
		print("FAIL zero values\n")
		return 1
	}
	slice := namedResultSlice()
	if len(slice) != 1 || slice[0] != 7 || namedResultString() != "named" {
		print("FAIL slice or string\n")
		return 1
	}
	print("PASS\n")
	return 0
}
