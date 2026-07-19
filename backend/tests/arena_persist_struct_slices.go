package main

type arenaPersistRow struct {
	A int
	B int
	C int
}

func renvo_runtime_ArenaMark() int             { return 0 }
func renvo_runtime_ArenaReset(mark int)        {}
func renvo_runtime_ArenaPersistMark() int      { return 0 }
func renvo_runtime_ArenaPersistReset(mark int) {}
func renvo_runtime_ArenaPersistCheckNameRefs(value []arenaPersistRow) []arenaPersistRow {
	return value
}
func renvo_runtime_ArenaPersistCheckBools(value []bool) []bool { return value }

func appMain() int {
	persist := renvo_runtime_ArenaPersistMark()
	mark := renvo_runtime_ArenaMark()
	rows := make([]arenaPersistRow, 3)
	rows[0] = arenaPersistRow{A: 11, B: 12, C: 13}
	rows[1] = arenaPersistRow{A: 21, B: 22, C: 23}
	rows[2] = arenaPersistRow{A: 31, B: 32, C: 33}
	flags := []bool{true, false, true, true}
	rows = renvo_runtime_ArenaPersistCheckNameRefs(rows)
	flags = renvo_runtime_ArenaPersistCheckBools(flags)
	renvo_runtime_ArenaReset(mark)
	if len(rows) != 3 || rows[0].A != 11 || rows[0].B != 12 || rows[0].C != 13 || rows[1].A != 21 || rows[1].B != 22 || rows[1].C != 23 || rows[2].A != 31 || rows[2].B != 32 || rows[2].C != 33 || len(flags) != 4 || !flags[0] || flags[1] || !flags[2] || !flags[3] {
		print("FAIL\n")
		return 1
	}
	renvo_runtime_ArenaPersistReset(persist)
	print("PASS\n")
	return 0
}
