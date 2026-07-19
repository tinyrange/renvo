package b

import "example.com/renvotests/extended/multipackage/case090/pkg/a"

func Value() int {
	return 24 + a.Value() - a.Value()
}
