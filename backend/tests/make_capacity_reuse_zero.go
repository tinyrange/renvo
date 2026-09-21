package main

func renvo_runtime_ArenaPersistMark() int      { return 0 }
func renvo_runtime_ArenaPersistReset(mark int) {}

type capacityStorage struct{ values []byte }

func allocateCapacity(s *capacityStorage) { s.values = make([]byte, 1, 8) }
func appMain(args []string) int {
	mark := renvo_runtime_ArenaPersistMark()
	for repeat := 0; repeat < 5; repeat++ {
		var s capacityStorage
		allocateCapacity(&s)
		full := s.values[:cap(s.values)]
		for i := 0; i < len(full); i++ {
			if full[i] != 0 {
				return 1
			}
			full[i] = 173
		}
		renvo_runtime_ArenaPersistReset(mark)
	}
	print("PASS\n")
	return 0
}
