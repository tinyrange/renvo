package b

import "example.com/renvotests/extended/multipackage/case057/pkg/a"

func Value() int {
	return 14 + a.Value() - a.Value()
}
