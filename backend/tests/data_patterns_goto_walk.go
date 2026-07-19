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
	var nodes []symbolRecord
	nodes = append(nodes, symbolRecord{value: 3, next: 1})
	nodes = append(nodes, symbolRecord{value: 4, next: -1})
	i := 0
	sum := 0
walk:
	if i < 0 {
		goto done
	}
	sum += nodes[i].value
	i = nodes[i].next
	goto walk
done:
	if sum != 7 {
		print("RENVO-0898 goto walk failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
