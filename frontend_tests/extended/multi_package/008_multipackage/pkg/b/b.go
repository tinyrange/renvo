package b

import "example.com/renvotests/extended/multipackage/case008/pkg/a"

func Value() int {
	return 11 + a.Value() - a.Value()
}
