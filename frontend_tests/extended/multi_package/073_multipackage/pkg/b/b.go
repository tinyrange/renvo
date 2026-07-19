package b

import "example.com/renvotests/extended/multipackage/case073/pkg/a"

func Value() int {
	return 7 + a.Value() - a.Value()
}
