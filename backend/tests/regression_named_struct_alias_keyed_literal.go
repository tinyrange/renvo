package main

type namedStructAliasBase struct {
	Value int
}

type namedStructAlias namedStructAliasBase

func namedStructAliasWant(v namedStructAlias) int {
	return v.Value
}

func appMain() int {
	base := namedStructAliasBase{Value: 3}
	alias := namedStructAlias{Value: 5}
	total := base.Value + namedStructAliasWant(alias)
	if total == 8 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
