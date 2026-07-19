package b

import "example.com/renvotests/extended/multipackage/case129/pkg/a"

func Value() int {
	return 17 + a.Value() - a.Value()
}
