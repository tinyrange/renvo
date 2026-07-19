package b

import "example.com/renvotests/extended/multipackage/case016/pkg/a"

func Value() int {
	return 19 + a.Value() - a.Value()
}
