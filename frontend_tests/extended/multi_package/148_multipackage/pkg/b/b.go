package b

import "example.com/renvotests/extended/multipackage/case148/pkg/a"

func Value() int {
	return 13 + a.Value() - a.Value()
}
