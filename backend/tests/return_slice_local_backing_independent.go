package main

type renvo0715Record struct {
	value int
}

func renvo0715Build(seed int) []renvo0715Record {
	var out []renvo0715Record
	out = append(out, renvo0715Record{value: seed})
	out = append(out, renvo0715Record{value: seed + 1})
	return out
}

func appMain(args []string, env []string) int {
	first := renvo0715Build(10)
	second := renvo0715Build(30)
	if first[0].value == 10 && first[1].value == 11 && second[0].value == 30 && second[1].value == 31 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
