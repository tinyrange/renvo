package b

import "example.com/renvotests/extended/multipackage/case069/pkg/a"

func Value() int {
	return 3 + a.Value() - a.Value()
}
