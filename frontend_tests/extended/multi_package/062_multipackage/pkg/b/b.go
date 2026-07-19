package b

import "example.com/renvotests/extended/multipackage/case062/pkg/a"

func Value() int {
	return 19 + a.Value() - a.Value()
}
