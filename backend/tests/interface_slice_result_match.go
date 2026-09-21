package main

type tree interface { Unwrap() []error }
type node struct{}
func (n node) Unwrap() []error { return nil }

func appMain(args []string) int {
	var value any = node{}
	matched, ok := value.(tree)
	if !ok || len(matched.Unwrap()) != 0 { return 1 }
	value = 3
	if _, ok := value.(tree); ok { return 2 }
	print("PASS\n")
	return 0
}
