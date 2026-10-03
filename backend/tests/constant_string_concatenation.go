package main

const stringConcatPrefix = "\x00" + "\x01"
const stringConcatTable = stringConcatPrefix + "\x02" + "\x02"
const stringConcatTyped string = (stringConcatPrefix + string("ab"))

func stringConcatLookup(x uint64) int { return int(stringConcatTable[x]) }
func stringConcatRead() string        { return stringConcatTyped }

func appMain(args []string) int {
	if stringConcatLookup(3) != 2 || len(stringConcatTable) != 4 || stringConcatTable[0] != 0 {
		print("FAIL lookup\n")
		return 1
	}
	if stringConcatRead() != "\x00\x01ab" || stringConcatTyped[2:] != "ab" {
		print("FAIL value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
