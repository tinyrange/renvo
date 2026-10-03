package main

type mediumRecord struct{ A, B, C, D, E int }

var mediumGlobal mediumRecord

func mediumCopy(v mediumRecord) mediumRecord {
	v.C += 10
	return v
}

func appMain(args []string) int {
	v := mediumRecord{1, 2, 3, 4, 5}
	mediumGlobal = v
	values := []mediumRecord{v, {6, 7, 8, 9, 10}}
	p := &values[1]
	*p = mediumCopy(mediumGlobal)
	w := mediumCopy(*p)
	if mediumGlobal.C != 3 || v.C != 3 || values[0].C != 3 || values[1].C != 13 || w.C != 23 || w.A != 1 || w.B != 2 || w.D != 4 || w.E != 5 {
		return 1
	}
	print("PASS\n")
	return 0
}
