package b

import "example.com/renvotests/extended/multipackage/case072/pkg/a"

func Value() int {
	return 6 + a.Value() - a.Value()
}
