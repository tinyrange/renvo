package b

import "example.com/renvotests/extended/multipackage/case082/pkg/a"

func Value() int {
	return 16 + a.Value() - a.Value()
}
