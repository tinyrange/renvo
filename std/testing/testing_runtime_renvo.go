//go:build renvo

package testing

import "os"

// RunTest is deliberately free of panic/recover until those constructs are
// lowered completely before compact units reach every backend.
func RunTest(name string, test func(*T)) bool {
	t := &T{name: name}
	test(t)
	if t.failed {
		printFailure(t, name)
	}
	return !t.failed
}

// FailNow cannot unwind only the current test without panic support. Exiting
// still preserves its defining guarantee that execution does not continue.
func (t *T) FailNow() {
	t.Fail()
	printFailure(t, t.name)
	os.Exit(1)
}
