package main

type renvoAppendEvaluationValue struct {
	a int
	b int
}

type renvoAppendEvaluationHolder struct {
	values []renvoAppendEvaluationValue
}

var renvoAppendEvaluationTarget renvoAppendEvaluationHolder
var renvoAppendEvaluationCalls int

func renvoAppendEvaluationGet() *renvoAppendEvaluationHolder {
	renvoAppendEvaluationCalls++
	return &renvoAppendEvaluationTarget
}

func appMain(args []string) int {
	renvoAppendEvaluationTarget.values = make([]renvoAppendEvaluationValue, 1)
	source := []renvoAppendEvaluationValue{{a: 1}, {a: 2}, {a: 3}}
	renvoAppendEvaluationGet().values = append(renvoAppendEvaluationGet().values, source...)
	if renvoAppendEvaluationCalls != 2 || len(renvoAppendEvaluationTarget.values) != 4 || renvoAppendEvaluationTarget.values[3].a != 3 {
		return 1
	}
	print("PASS\n")
	return 0
}
