package main

type resultRecord struct {
	key   int
	value int
	next  *resultRecord
}

func makeResultRecord(n int) resultRecord {
	return resultRecord{key: n, value: n * n}
}

func appMain() int {
	r := new(resultRecord)
	*r = makeResultRecord(7)
	if r.key != 7 || r.value != 49 || r.next != nil {
		print("FAIL aggregate result parameter\n")
		return 1
	}
	print("PASS\n")
	return 0
}
