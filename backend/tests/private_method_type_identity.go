package main

type sourceValueType int

func (v sourceValueType) secret() int { return int(v) }

type targetContract interface{ secret() int }

func targetMatches(value any) bool {
	_, ok := value.(targetContract)
	return ok
}

func appMain(args []string) int {
	wantSame := len(args) < 2 || args[1] != "distinct"
	if targetMatches(sourceValueType(42)) != wantSame {
		return 1
	}
	print("PASS\n")
	return 0
}
