package main

func renvo_runtime_ArenaMark() int      { return 0 }
func renvo_runtime_ArenaReset(mark int) {}

func appMain() int {
	for n := 0; n < 33; n++ {
		mark := renvo_runtime_ArenaMark()
		old := make([]byte, n+7)
		for i := 0; i < len(old); i++ {
			old[i] = 173
		}
		renvo_runtime_ArenaReset(mark)
		values := make([]byte, n)
		for i := 0; i < len(values); i++ {
			if values[i] != 0 {
				return 1
			}
		}
		for i := n; i < len(old); i++ {
			if old[i] != 173 {
				return 2
			}
		}
	}
	print("PASS\n")
	return 0
}
