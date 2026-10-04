package main

type embeddedRecord struct{ Value int }
type aliasRecord = struct{ embeddedRecord }
type otherRecord = struct{ Different embeddedRecord }
type explicitRecord = struct{ embeddedRecord embeddedRecord }
type taggedRecord = struct {
	Value int `json:"value"`
}
type sameTaggedRecord = struct {
	Value int "json:\"value\""
}
type differentTaggedRecord = struct {
	Value int `json:"other"`
}
type definedRecord struct{ embeddedRecord }

func acceptAnonymous(value any) bool {
	type localRecord = struct{ embeddedRecord }
	_, ok := value.(localRecord)
	return ok
}
func rejectAnonymous(value any) bool {
	_, other := value.(otherRecord)
	_, explicit := value.(explicitRecord)
	_, defined := value.(definedRecord)
	return other || explicit || defined
}
func acceptPointer(value any) bool {
	type localRecord = struct{ embeddedRecord }
	_, ok := value.(*localRecord)
	return ok
}
func checkTags(value any) bool {
	_, same := value.(sameTaggedRecord)
	_, different := value.(differentTaggedRecord)
	return same && !different
}
func appMain(args []string) int {
	value := aliasRecord{embeddedRecord{42}}
	if !acceptAnonymous(value) || rejectAnonymous(value) || !acceptPointer(&value) || !checkTags(taggedRecord{42}) {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
