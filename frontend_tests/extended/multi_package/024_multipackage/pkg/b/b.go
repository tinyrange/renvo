package b

import "example.com/renvotests/extended/multipackage/case024/pkg/a"

func Value() int {
	return 4 + a.Value() - a.Value()
}
