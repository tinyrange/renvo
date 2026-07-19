package main

type renvoWasmReturnStrideRecord struct {
	value int
}

func renvoWasmReturnStrideBuild(seed int) []renvoWasmReturnStrideRecord {
	var out []renvoWasmReturnStrideRecord
	out = append(out, renvoWasmReturnStrideRecord{value: seed})
	out = append(out, renvoWasmReturnStrideRecord{value: seed + 1})
	return out
}

func appMain(args []string, env []string) int {
	values := renvoWasmReturnStrideBuild(10)
	if values[0].value == 10 && values[1].value == 11 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
