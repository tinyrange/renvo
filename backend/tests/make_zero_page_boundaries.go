package main

func renvo_runtime_ArenaMark() int      { return 0 }
func renvo_runtime_ArenaReset(mark int) {}

func appMain() int {
	sizes := []int{0, 1, 4095, 4096, 8191, 8192, 8193, 16384}
	offsets := []int{0, 1, 15, 4095}
	for _, size := range sizes {
		for _, offset := range offsets {
			outer := renvo_runtime_ArenaMark()
			prefix := make([]byte, offset+1)
			for i := 0; i < len(prefix); i++ {
				prefix[i] = 197
			}
			mark := renvo_runtime_ArenaMark()
			for repeat := 0; repeat < 2; repeat++ {
				values := make([]byte, 0, size)
				values = values[:cap(values)]
				for i := 0; i < len(values); i++ {
					if values[i] != 0 {
						return 1
					}
					values[i] = 173
				}
				for i := 0; i < len(prefix); i++ {
					if prefix[i] != 197 {
						return 2
					}
				}
				renvo_runtime_ArenaReset(mark)
			}
			renvo_runtime_ArenaReset(outer)
		}
	}
	print("PASS\n")
	return 0
}
