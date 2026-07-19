package main

func renvo_runtime_ArenaMark() int { return 0 }

func renvo_runtime_ArenaReset(mark int) {}

func renvo_runtime_ArenaPersistString(value string) string { return value }

type persistedResult struct {
	path string
	code int
}

func dynamicPersistedPath() string {
	data := make([]byte, 0, 4)
	data = append(data, 'P')
	data = append(data, 'A')
	data = append(data, 'S')
	data = append(data, 'S')
	return string(data)
}

func persistResult(path string) persistedResult {
	var result persistedResult
	result.path = renvo_runtime_ArenaPersistString(path)
	result.code = 7
	return result
}

func appMain(args []string) int {
	mark := renvo_runtime_ArenaMark()
	result := persistResult(dynamicPersistedPath())
	renvo_runtime_ArenaReset(mark)
	if result.path != "PASS" || result.code != 7 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
