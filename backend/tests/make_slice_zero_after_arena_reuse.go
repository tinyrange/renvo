package main

func renvo_runtime_ArenaMark() int { return 0 }

func renvo_runtime_ArenaReset(mark int) {}

func dynamicMakeLength() int { return 4 }

func appMain(args []string) int {
	mark := renvo_runtime_ArenaMark()
	dirty := make([]int, dynamicMakeLength())
	for i := 0; i < len(dirty); i++ {
		dirty[i] = i + 1
	}
	renvo_runtime_ArenaReset(mark)

	clean := make([]int, dynamicMakeLength())
	for i := 0; i < len(clean); i++ {
		if clean[i] != 0 {
			print("FAIL\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
