package b

import "example.com/renvotests/extended/multipackage/case035/pkg/a"

func Value() int {
	return 15 + a.Value() - a.Value()
}
