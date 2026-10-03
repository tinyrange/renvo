package main

type sourceRecord struct {
	Value int "old"
	Items [2]int
}

type targetRecord struct {
	Value int "new"
	Items [2]int
}

var sourceGlobal = sourceRecord{42, [2]int{7, 9}}

func convertedRecord(v sourceRecord) targetRecord { return targetRecord(v) }

func appMain(args []string) int {
	local := sourceGlobal
	converted := convertedRecord(local)
	direct := targetRecord(sourceRecord{21, [2]int{3, 4}})
	global := targetRecord(sourceGlobal)
	items := []sourceRecord{local}
	indexed := targetRecord(items[0])
	pointer := &local
	indirect := targetRecord(*pointer)
	local.Items[0] = 99
	if converted.Value != 42 || converted.Items[0] != 7 || direct.Value != 21 || direct.Items[1] != 4 || global.Value != 42 || indexed.Items[1] != 9 || indirect.Items[0] != 7 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
