package main

type Flow struct {
	Next, Alternate uint64
	Conditional     bool
}
type ClosureEnv struct {
	Root     *uint64
	Attempts *int
	Best     *float64
}

func search(env *ClosureEnv, flow Flow, counts [2]uint64, probability float64) float64 {
	targets := []uint64{flow.Next, flow.Alternate}
	result := float64(0)
	for _, target := range targets {
		(*env.Attempts)++
		reach := probability
		if flow.Conditional && counts[0]+counts[1] != 0 {
			edge := 0
			if target == flow.Alternate {
				edge = 1
			}
			reach *= float64(counts[edge]+1) / float64(counts[0]+counts[1]+2)
		}
		if target == *env.Root && reach < 1 {
			result = reach
		}
	}
	return result
}
func appMain(args []string) int {
	root, attempts, best := uint64(7), 0, float64(0)
	env := &ClosureEnv{Root: &root, Attempts: &attempts, Best: &best}
	for _, counts := range [][2]uint64{{0, 256}, {256, 0}, {7, 3}} {
		observed := search(env, Flow{Next: 3, Alternate: 7, Conditional: true}, counts, 1)
		if observed <= 0 || observed >= 1 {
			print("FAIL ratio\n")
			return 1
		}
	}
	if attempts != 6 {
		print("FAIL attempts\n")
		return 1
	}
	print("PASS\n")
	return 0
}
