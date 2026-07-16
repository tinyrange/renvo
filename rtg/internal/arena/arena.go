package arena

func rtg_runtime_ArenaMark() int { return 0 }

func rtg_runtime_ArenaReset(mark int) {}

func rtg_runtime_ArenaPersistMark() int { return 0 }

func rtg_runtime_ArenaPersistReset(mark int) {}

func rtg_runtime_ArenaPersistString(value string) string { return value }

func rtg_runtime_ArenaPersistBytes(value []byte) []byte { return value }

func rtg_runtime_ArenaDiscard(start int, end int) {}

func Mark() int { return rtg_runtime_ArenaMark() }

func Reset(mark int) {
	end := rtg_runtime_ArenaMark()
	rtg_runtime_ArenaDiscard(mark, end)
	rtg_runtime_ArenaReset(mark)
}

func PersistMark() int { return rtg_runtime_ArenaPersistMark() }

func PersistReset(mark int) { rtg_runtime_ArenaPersistReset(mark) }

func PersistString(value string) string { return rtg_runtime_ArenaPersistString(value) }

func PersistBytes(value []byte) []byte { return rtg_runtime_ArenaPersistBytes(value) }

// Discard releases complete pages wholly contained in a dead arena range
// without rewinding the allocator or invalidating later allocations.
func Discard(start int, end int) { rtg_runtime_ArenaDiscard(start, end) }
