package main

type MethodInfo struct {
	name    string
	pointer bool
}

type MethodEntry struct {
	name string
	info MethodInfo
}

type MethodTable []MethodEntry

func (table MethodTable) set(name string, info MethodInfo) MethodTable {
	for i := 0; i < len(table); i++ {
		if table[i].name == name {
			table[i].info = info
			return table
		}
	}
	return append(table, MethodEntry{name: name, info: info})
}

func fill(types *MethodTable) {
	info := MethodInfo{name: "string"}
	*types = types.set("args", info)
}

func appMain(args []string, env []string) int {
	var types MethodTable
	fill(&types)
	if len(types) != 1 {
		print("bad length\n")
		return 1
	}
	if types[0].name != "args" {
		print("bad entry name\n")
		return 1
	}
	if types[0].info.name != "string" {
		print("bad info name\n")
		return 1
	}
	print("PASS\n")
	return 0
}
