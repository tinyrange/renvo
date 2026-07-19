package main

type info struct {
	qualifier string
	name      string
	pointer   bool
}

type entry struct {
	name string
	info info
}

func lookup(table []entry, name string) info {
	for i := 0; i < len(table); i++ {
		item := table[i]
		if item.name == name {
			return item.info
		}
	}
	return info{}
}

func appMain() int {
	table := []entry{{name: "want", info: info{qualifier: "PASS", name: "\n", pointer: true}}}
	got := lookup(table, "want")
	if got.pointer && got.qualifier == "PASS" && got.name == "\n" {
		print(got.qualifier)
		print(got.name)
		return 0
	}
	print("FAIL\n")
	return 1
}
