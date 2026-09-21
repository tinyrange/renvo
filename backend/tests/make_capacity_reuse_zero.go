package main

func renvo_runtime_ArenaMark() int      { return 0 }
func renvo_runtime_ArenaReset(mark int) {}

type capacityStorage struct{ values []byte }

func allocateCapacity(s *capacityStorage)                      { s.values = make([]byte, 1, 8) }
func allocateDynamicCapacity(s *capacityStorage, capacity int) { s.values = make([]byte, 1, capacity) }
func appMain(args []string) int {
	mark := renvo_runtime_ArenaMark()
	for repeat := 0; repeat < 10; repeat++ {
		var s capacityStorage
		if repeat < 5 {
			allocateCapacity(&s)
		} else {
			allocateDynamicCapacity(&s, 8)
		}
		full := s.values[:cap(s.values)]
		for i := 0; i < len(full); i++ {
			if full[i] != 0 {
				return 1
			}
			full[i] = 173
		}
		renvo_runtime_ArenaReset(mark)
	}
	print("PASS\n")
	return 0
}
