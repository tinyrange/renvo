package b

import "example.com/renvotests/extended/multipackage/case119/pkg/a"

func Value() int {
	return 7 + a.Value() - a.Value()
}
