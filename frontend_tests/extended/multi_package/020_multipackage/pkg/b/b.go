package b

import "example.com/renvotests/extended/multipackage/case020/pkg/a"

func Value() int {
	return 23 + a.Value() - a.Value()
}
