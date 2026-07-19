package main

type arraySlicingRecord struct {
	value int
}

type arraySlicingNamed [3]int

type arraySlicingContainer struct {
	values [2]int
}

var arraySlicingGlobal = [4]int{1, 2, 3, 4}

func appMain(args []string) int {
	values := [3]int{1, 2, 3}
	part := values[1:]
	part[0] = 9
	if len(part) != 2 || cap(part) != 2 || values[1] != 9 {
		return 1
	}

	full := values[0:2:2]
	if len(full) != 2 || cap(full) != 2 {
		return 1
	}

	globalPart := arraySlicingGlobal[1:3]
	globalPart[1] = 10
	if len(globalPart) != 2 || cap(globalPart) != 3 || arraySlicingGlobal[2] != 10 {
		return 1
	}

	strings := [2]string{"a", "b"}
	stringPart := strings[:]
	stringPart[0] = "changed"
	if stringPart[0] != "changed" {
		return 1
	}

	pointer := &values
	pointerPart := pointer[1:2:3]
	pointerPart[0] = 7
	if len(pointerPart) != 1 || cap(pointerPart) != 2 || values[1] != 7 {
		return 1
	}
	dereferencedPart := (*pointer)[0:1]
	dereferencedPart[0] = 11
	if values[0] != 11 {
		return 1
	}

	records := [2]arraySlicingRecord{{value: 3}, {value: 4}}
	recordPart := records[:]
	recordPart[1].value = 8
	if recordPart[1].value != 8 {
		return 1
	}

	nested := [2][2]int{{1, 2}, {3, 4}}
	nestedPart := nested[1:]
	nestedPart[0][1] = 6
	if nestedPart[0][1] != 6 {
		return 1
	}
	innerPart := nested[1][:]
	innerPart[0] = 12
	if nestedPart[0][0] != 12 {
		return 1
	}

	named := arraySlicingNamed{1, 2, 3}
	namedPart := named[1:]
	namedPart[0] = 13
	if named[1] != 13 {
		return 1
	}

	container := arraySlicingContainer{values: [2]int{4, 5}}
	fieldPart := container.values[:]
	fieldPart[1] = 14
	if container.values[1] != 14 {
		return 1
	}

	var slices [2][]int
	var maps [2]map[string]int
	var interfaces [2]interface{}
	var functions [2]func(int) int
	if len(slices[:]) != 2 || len(maps[:]) != 2 || len(interfaces[:]) != 2 || len(functions[:]) != 2 {
		return 1
	}

	print("PASS\n")
	return 0
}
