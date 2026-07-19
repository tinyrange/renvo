package b

import "example.com/renvotests/extended/multipackage/case038/pkg/a"

func Value() int {
	return 18 + a.Value() - a.Value()
}
