package main

type fixedRecord struct{ A, B, C, D int }

var fixedGlobal fixedRecord

func fixedIdentity(v fixedRecord) fixedRecord { return v }

func appMain(args []string) int {
	v := fixedRecord{1, 2, 3, 4}
	fixedGlobal = v
	w := fixedIdentity(fixedGlobal)
	p := &w
	*p = *p
	values := []fixedRecord{v, {5, 6, 7, 8}}
	values[1] = *p
	*p = values[1]
	if w.A != 1 || w.B != 2 || w.C != 3 || w.D != 4 || values[1].D != 4 || fixedGlobal.C != 3 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
