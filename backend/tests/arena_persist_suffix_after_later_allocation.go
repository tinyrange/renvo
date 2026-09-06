package main

func renvo_runtime_ArenaMark() int                         { return 0 }
func renvo_runtime_ArenaReset(mark int)                    {}
func renvo_runtime_ArenaPersistReset(mark int)             {}
func renvo_runtime_ArenaPersistString(value string) string { return value }
func renvo_runtime_ArenaBytesStart(value []byte) int       { return 0 }

func appMain() int {
	mark := renvo_runtime_ArenaMark()
	padding := make([]byte, 16)
	padding[0] = 1
	unit := make([]byte, 64)
	unit[0] = 'R'
	unit[1] = 'N'
	unit[2] = 'V'
	unit[3] = 'O'
	// A later allocation makes the low-arena cursor unsuitable for deriving
	// the unit's address from its capacity.
	later := make([]byte, 37)
	for i := 0; i < len(later); i++ {
		later[i] = byte(i + 1)
	}
	start := renvo_runtime_ArenaBytesStart(unit)
	start -= start % 16
	renvo_runtime_ArenaPersistReset(start)
	renvo_runtime_ArenaReset(mark)
	kept := renvo_runtime_ArenaPersistString("persistent handoff metadata must not overwrite the promoted result bytes")
	if kept != "persistent handoff metadata must not overwrite the promoted result bytes" ||
		unit[0] != 'R' || unit[1] != 'N' || unit[2] != 'V' || unit[3] != 'O' {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
