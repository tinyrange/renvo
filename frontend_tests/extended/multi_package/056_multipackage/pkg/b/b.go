package b

import "example.com/renvotests/extended/multipackage/case056/pkg/a"

func Value() int {
	return 13 + a.Value() - a.Value()
}
