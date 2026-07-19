package b

import "example.com/renvotests/extended/multipackage/case066/pkg/a"

func Value() int {
	return 23 + a.Value() - a.Value()
}
