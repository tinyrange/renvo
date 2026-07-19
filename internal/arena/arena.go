package arena

func renvo_runtime_ArenaMark() int { return 0 }

func renvo_runtime_ArenaReset(mark int) {}

func renvo_runtime_ArenaPersistMark() int { return 0 }

func renvo_runtime_ArenaPersistReset(mark int) {}

func renvo_runtime_ArenaPersistString(value string) string { return value }

func renvo_runtime_ArenaPersistBytes(value []byte) []byte { return value }

func renvo_runtime_ArenaDiscard(start int, end int) {}

func renvo_runtime_ArenaDiscardBytes(value []byte) {}

func Mark() int { return renvo_runtime_ArenaMark() }

func Reset(mark int) {
	end := renvo_runtime_ArenaMark()
	renvo_runtime_ArenaDiscard(mark, end)
	renvo_runtime_ArenaReset(mark)
}

func PersistMark() int { return renvo_runtime_ArenaPersistMark() }

func PersistReset(mark int) { renvo_runtime_ArenaPersistReset(mark) }

func PersistString(value string) string { return renvo_runtime_ArenaPersistString(value) }

func PersistBytes(value []byte) []byte { return renvo_runtime_ArenaPersistBytes(value) }

// Discard releases complete pages wholly contained in a dead arena range
// without rewinding the allocator or invalidating later allocations.
func Discard(start int, end int) { renvo_runtime_ArenaDiscard(start, end) }

// DiscardBytes releases complete pages covered by a dead byte slice without
// changing the arena allocation cursor. Callers must not read value again.
func DiscardBytes(value []byte) { renvo_runtime_ArenaDiscardBytes(value) }
