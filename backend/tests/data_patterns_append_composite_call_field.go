package main

type appendCompositeCallFieldRecord struct {
	value int
	tag   int
}

func appendCompositeCallFieldValue(v int) int {
	return v + 4
}

func appMain(args []string) int {
	var records []appendCompositeCallFieldRecord
	records = append(records, appendCompositeCallFieldRecord{value: appendCompositeCallFieldValue(5), tag: 2})
	if len(records) != 1 || records[0].value != 9 || records[0].tag != 2 {
		print("append composite call field failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
