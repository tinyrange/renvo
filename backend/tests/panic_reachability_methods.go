package main

type Runner interface{ Run() int }
type Good struct{ value int }
type Bad struct{ value int }

func (v Good) Run() int { return v.value }
func (v Bad) Run() int  { panic("unused method") }
func appMain(args []string) int {
	var runner Runner = Good{value: 7}
	if runner.Run() != 7 {
		return 1
	}
	print("PASS\n")
	return 0
}
