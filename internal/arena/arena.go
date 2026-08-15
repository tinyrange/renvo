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

// Rewind makes short-lived storage reusable without returning its pages to the
// operating system. Tight compiler loops use it when the same scratch high-water
// mark will be touched again; the enclosing phase still calls Reset to release
// those pages once the translation unit is complete.
func Rewind(mark int) { renvo_runtime_ArenaReset(mark) }

func PersistMark() int { return renvo_runtime_ArenaPersistMark() }

func PersistReset(mark int) { renvo_runtime_ArenaPersistReset(mark) }

func PersistString(value string) string { return renvo_runtime_ArenaPersistString(value) }

func PersistBytes(value []byte) []byte { return renvo_runtime_ArenaPersistBytes(value) }

// PersistLastBytes persists the most recent low-arena byte allocation. It uses
// the ordinary copy when both allocations fit; otherwise it transfers arena
// ownership in place. Callers must pass the low-arena mark that may be restored
// and must not retain any other allocation made after that mark. Reserving a
// few alignment bytes below the slice keeps promotion valid for every native
// word size.
func PersistLastBytes(value []byte, lowMark int) []byte {
	end := renvo_runtime_ArenaMark()
	if end == 0 || len(value) == 0 {
		return value
	}
	persistEnd := renvo_runtime_ArenaPersistMark()
	if persistEnd-end >= len(value) {
		return renvo_runtime_ArenaPersistBytes(value)
	}
	start := end - cap(value)
	start -= start % 16
	if start < lowMark {
		return renvo_runtime_ArenaPersistBytes(value)
	}
	renvo_runtime_ArenaPersistReset(start)
	renvo_runtime_ArenaReset(lowMark)
	return value
}

// Discard releases complete pages wholly contained in a dead arena range
// without rewinding the allocator or invalidating later allocations.
func Discard(start int, end int) { renvo_runtime_ArenaDiscard(start, end) }

// DiscardBytes releases complete pages covered by a dead byte slice without
// changing the arena allocation cursor. Callers must not read value again.
func DiscardBytes(value []byte) { renvo_runtime_ArenaDiscardBytes(value) }
