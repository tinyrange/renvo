package b

import "example.com/renvotests/extended/multipackage/case005/pkg/a"

func Value() int {
	return 8 + a.Value() - a.Value()
}
