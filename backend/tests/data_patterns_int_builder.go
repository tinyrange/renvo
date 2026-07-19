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
	var buf []int
	i := 1
	for i < 5 {
		buf = append(buf, i*i)
		i += 1
	}
	if len(buf) != 4 || buf[2] != 9 {
		print("RENVO-0882 int builder failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
