package main

type Runner interface{ Run() int }
type Good struct{ value int }
type Bad struct{ value int }

func (v Good) Run() int { return v.value }
func (v Bad) Run() int  { panic("unused method") }
func good() int         { return 5 }
func bad() int          { panic("unused function value candidate") }

var selected func() int = good

func appMain(args []string) int {
	var runner Runner = Good{value: 7}
	if selected() != 5 || runner.Run() != 7 {
		return 1
	}
	print("PASS\n")
	return 0
}
