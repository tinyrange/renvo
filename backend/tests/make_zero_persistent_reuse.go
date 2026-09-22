package main

func renvo_runtime_ArenaPersistMark() int      { return 0 }
func renvo_runtime_ArenaPersistReset(mark int) {}

func appMain() int {
	prefix := new([17]byte)
	for i := 0; i < len(prefix); i++ {
		prefix[i] = 197
	}
	mark := renvo_runtime_ArenaPersistMark()
	for repeat := 0; repeat < 3; repeat++ {
		values := new([16385]byte)
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
		renvo_runtime_ArenaPersistReset(mark)
	}
	print("PASS\n")
	return 0
}
