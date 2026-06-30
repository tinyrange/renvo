package main

func rtg_runtime_ArenaMark() int {
	return 0
}

func rtg_runtime_ArenaReset(mark int) {
}

func rtg_runtime_ArenaPersistBytes(value []byte) []byte {
	return value
}

func persistBytes(value []byte) []byte {
	return rtg_runtime_ArenaPersistBytes(value)
}

func makeBytes() []byte {
	data := make([]byte, 0, 1500000)
	for i := 0; i < 1500000; i++ {
		data = append(data, byte('A'+i%26))
	}
	data[0] = 'P'
	data[1] = 'A'
	data[2] = 'S'
	data[3] = 'S'
	data[1499999] = '\n'
	return data
}

func appMain(args []string, env []string) int {
	mark := rtg_runtime_ArenaMark()
	data := persistBytes(makeBytes())
	rtg_runtime_ArenaReset(mark)
	if len(data) != 1500000 {
		return 1
	}
	if data[0] != 'P' {
		return 1
	}
	if data[1] != 'A' {
		return 1
	}
	if data[2] != 'S' {
		return 1
	}
	if data[3] != 'S' {
		return 1
	}
	if data[1499999] != '\n' {
		return 1
	}
	print("PASS\n")
	return 0
}
