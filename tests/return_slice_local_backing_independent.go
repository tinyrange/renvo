package main

type rtg0715Record struct {
	value int
}

func rtg0715Build(seed int) []rtg0715Record {
	var out []rtg0715Record
	out = append(out, rtg0715Record{value: seed})
	out = append(out, rtg0715Record{value: seed + 1})
	return out
}

func appMain(args []string, env []string) int {
	first := rtg0715Build(10)
	second := rtg0715Build(30)
	if first[0].value == 10 && first[1].value == 11 && second[0].value == 30 && second[1].value == 31 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
