package main

type symbolRecord struct {
	name     string
	value    int
	tag      byte
	marked   bool
	children []int
	next     int
	parent   int
}

var globalRecords []symbolRecord

func findSymbol(list []symbolRecord, name string) int {
	i := 0
	for i < len(list) {
		if list[i].name == name {
			return i
		}
		i += 1
	}
	return -1
}
func walkNext(list []symbolRecord, at int) int {
	if at < 0 {
		return 0
	}
	return list[at].value + walkNext(list, list[at].next)
}
func current(list []symbolRecord, i int) *symbolRecord { return &list[i] }

func markDepth(n int) bool {
	if n == 0 {
		return true
	}
	return markDepth(n - 1)
}
func appendRecord(table []symbolRecord, value int) []symbolRecord {
	return append(table, symbolRecord{value: value})
}

func appMain(args []string) int {
	var records []symbolRecord
	records = append(records, symbolRecord{name: "use", next: -1})
	records = append(records, symbolRecord{name: "target", value: 44})
	i := findSymbol(records, "target")
	records[0].next = i
	if records[records[0].next].value != 44 {
		print("RENVO-0892 forward resolve failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
