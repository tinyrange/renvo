package b

import "example.com/renvotests/extended/multipackage/case033/pkg/a"

func Value() int {
	return 13 + a.Value() - a.Value()
}
