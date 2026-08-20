package main

type renvoObjectSmallAggregateResult struct {
	magnitude uint32
	shiftOne uint8
	shiftTwo uint8
}

//export renvo_object_small_aggregate_export_return
func renvoObjectSmallAggregateExportReturn(value uint32) renvoObjectSmallAggregateResult {
	return renvoObjectSmallAggregateResult{
		magnitude: value + 9,
		shiftOne:  3,
		shiftTwo:  5,
	}
}

func appMain(args []string) int {
	result := renvoObjectSmallAggregateExportReturn(23)
	if result.magnitude != 32 || result.shiftOne != 3 || result.shiftTwo != 5 {
		print("object small aggregate export return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
