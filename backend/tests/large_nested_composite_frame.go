package main

type Prefix struct {
	Clock     *uint64
	Padding   [8400]uint64
	Remaining uint64
}
type Context struct {
	Value Prefix
	Owner *uint64
}

func appMain(args []string) int {
	epoch := uint64(7)
	memory := Context{Value: Prefix{Clock: &epoch, Remaining: 9}}
	if memory.Value.Remaining != 9 || *memory.Value.Clock != 7 {
		return 1
	}
	print("PASS\n")
	return 0
}
