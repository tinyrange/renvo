package b

import "example.com/renvotests/extended/multipackage/case041/pkg/a"

func Value() int {
	return 21 + a.Value() - a.Value()
}
