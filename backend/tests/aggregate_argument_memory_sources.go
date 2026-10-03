package main

type argumentRecord struct{ A, B, C, D, E, F, G, H int }
type argumentHolder struct {
	Prefix int
	Record argumentRecord
}

var argumentGlobal argumentRecord

func argumentCheck(prefix int, value argumentRecord, suffix int) bool {
	if prefix != 10 || suffix != 20 || value.A != 1 || value.B != 2 || value.C != 3 || value.D != 4 || value.E != 5 || value.F != 6 || value.G != 7 || value.H != 8 {
		return false
	}
	value.A = 99
	return true
}

func appMain(args []string) int {
	r := argumentRecord{1, 2, 3, 4, 5, 6, 7, 8}
	argumentGlobal = r
	p := &r
	values := []argumentRecord{r}
	holder := argumentHolder{Record: r}
	if !argumentCheck(10, r, 20) || !argumentCheck(10, argumentGlobal, 20) || !argumentCheck(10, *p, 20) || !argumentCheck(10, values[0], 20) || !argumentCheck(10, holder.Record, 20) {
		print("FAIL\n")
		return 1
	}
	if r.A != 1 || argumentGlobal.A != 1 || values[0].A != 1 || holder.Record.A != 1 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
