package b

import "example.com/renvotests/extended/multipackage/case122/pkg/a"

func Value() int {
	return 10 + a.Value() - a.Value()
}
