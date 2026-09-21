package main

type fault struct{}
func (f *fault) Error() string { return "fault" }
type wrong struct{}
func (w wrong) Error() int { return 1 }

func appMain(args []string) int {
	var value any = 3
	if _, ok := value.(error); ok { return 1 }
	value = wrong{}
	if _, ok := value.(error); ok { return 2 }
	value = fault{}
	if _, ok := value.(error); ok { return 3 }
	var nilFault *fault
	value = nilFault
	if _, ok := value.(error); !ok { return 4 }
	var target error
	value = &target
	if _, ok := value.(*error); !ok { return 5 }
	if _, ok := value.(*any); ok { return 6 }
	print("PASS\n")
	return 0
}
