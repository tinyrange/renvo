package b

import "example.com/renvotests/extended/multipackage/case091/pkg/a"

func Value() int {
	return 25 + a.Value() - a.Value()
}
