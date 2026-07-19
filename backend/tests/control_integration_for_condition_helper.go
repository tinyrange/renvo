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
	const wanted = 3
	globalStep = 0
	count := 0
	for nextStep() {
		count += 1
	}
	if count != wanted {
		print("RENVO-0921 for condition helper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
