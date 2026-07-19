package main

type renvo0832Entry struct {
	key   int
	value int
}

func appMain(args []string) int {
	var entries []renvo0832Entry
	entries = append(entries, renvo0832Entry{key: 1, value: 10})
	entries = append(entries, renvo0832Entry{key: 2, value: 20})
	found := 0
	i := 0
	for i < len(entries) {
		if entries[i].key == 2 {
			found = entries[i].value
		}
		i = i + 1
	}
	if found != 20 {
		print("RENVO-0832 linear lookup failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
