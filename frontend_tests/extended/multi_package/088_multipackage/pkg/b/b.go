package b

import "example.com/renvotests/extended/multipackage/case088/pkg/a"

func Value() int {
	return 22 + a.Value() - a.Value()
}
