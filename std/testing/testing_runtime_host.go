//go:build !renvo

package testing

type failNow struct{}

func RunTest(name string, test func(*T)) bool {
	t := &T{name: name}
	func() {
		defer func() {
			if recover() != nil {
				t.Fail()
			}
		}()
		test(t)
	}()
	if t.failed {
		printFailure(t, name)
	}
	return !t.failed
}

func (t *T) FailNow() {
	t.Fail()
	panic(failNow{})
}
