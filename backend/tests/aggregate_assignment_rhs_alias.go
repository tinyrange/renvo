package main

type aggregateAssignmentValue struct {
	value uint64
}

func aggregateAssignmentRead(value aggregateAssignmentValue) uint64 {
	return value.value
}

func aggregateAssignmentPreserve(value aggregateAssignmentValue) uint64 {
	value = aggregateAssignmentValue{value: aggregateAssignmentRead(value)}
	return value.value
}

func appMain(args []string) int {
	if aggregateAssignmentPreserve(aggregateAssignmentValue{value: 2}) != 2 {
		return 1
	}
	print("PASS\n")
	return 0
}
