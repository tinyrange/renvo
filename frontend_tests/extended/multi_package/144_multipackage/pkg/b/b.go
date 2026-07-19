package b

import "example.com/renvotests/extended/multipackage/case144/pkg/a"

func Value() int {
	return 9 + a.Value() - a.Value()
}
