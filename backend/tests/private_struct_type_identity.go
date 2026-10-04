package main

type sourceRecord = struct{ value int }

func sourceValue() any { return sourceRecord{value: 42} }

type targetRecord = struct{ value int }

func targetMatches(value any) bool {
	_, ok := value.(targetRecord)
	return ok
}

func appMain(args []string) int {
	// Source compilation has one package. The compact-unit regression also
	// gives these declarations different package owners and passes "distinct".
	wantSame := len(args) < 2 || args[1] != "distinct"
	if targetMatches(sourceValue()) != wantSame {
		return 1
	}
	print("PASS\n")
	return 0
}
