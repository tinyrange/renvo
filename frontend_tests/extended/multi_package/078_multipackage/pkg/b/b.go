package b

import "example.com/renvotests/extended/multipackage/case078/pkg/a"

func Value() int {
	return 12 + a.Value() - a.Value()
}
