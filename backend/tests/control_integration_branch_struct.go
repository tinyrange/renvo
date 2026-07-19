package main

type flowState struct {
	value int
	count int
	buf   []int
}

var globalStep int = 0

func decUntil(n int) int {
	sum := 0
	for n > 0 {
		sum += n
		n -= 1
	}
	return sum
}
func nextStep() bool { globalStep += 1; return globalStep < 4 }
func status(v int) int {
	if v > 4 {
		return 1
	}
	return 0
}
func recurseLoop(n int) int {
	if n == 0 {
		return 0
	}
	i := 0
	sum := n
	for i < n {
		sum += i
		i += 1
	}
	return sum + recurseLoop(n-1)
}

func decThenRecurse(n int) int {
	total := 0
	for n > 2 {
		total += n
		n -= 1
	}
	return total + decUntil(n)
}
func choosePointer(v bool) bool    { return v }
func ptrSum(p *int, extra int) int { return *p + extra }
func machineA(v int) int           { return machineB(v + 3) }
func machineB(v int) int           { return v*2 + 4 }

func appMain(args []string) int {
	state := flowState{value: 2, count: 1}
	if len(args) > 0 {
		state = flowState{value: 5, count: 3}
	}
	if state.value+state.count != 8 {
		print("RENVO-0914 branch struct failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
