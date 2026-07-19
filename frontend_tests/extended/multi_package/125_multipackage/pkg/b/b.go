package b

import "example.com/renvotests/extended/multipackage/case125/pkg/a"

func Value() int {
	return 13 + a.Value() - a.Value()
}
