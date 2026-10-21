package main

func renvo_runtime_ArenaMark() int { return 0 }

const shadowedStringTree = "package"

var initializedStringTree = "head" + ("\x00" + "é") + `tail`
var stringTreeCalls int

func literalStringTree() string {
	return "head" + ("\x00" + "é") + `tail`
}

func stringTreePart(n int) string {
	stringTreeCalls = stringTreeCalls*10 + n
	if n == 1 {
		return "one"
	}
	return "two"
}

func appMain(args []string) int {
	before := renvo_runtime_ArenaMark()
	value := literalStringTree()
	after := renvo_runtime_ArenaMark()
	if before != after || value != "head\x00étail" || initializedStringTree != value {
		print("FAIL literal storage\n")
		return 1
	}
	shadowedStringTree := "local"
	if "["+shadowedStringTree+"]" != "[local]" {
		print("FAIL shadowing\n")
		return 1
	}
	joined := "(" + stringTreePart(1) + ":" + stringTreePart(2) + ")"
	if joined != "(one:two)" || stringTreeCalls != 12 {
		print("FAIL evaluation order\n")
		return 1
	}
	print("PASS\n")
	return 0
}
