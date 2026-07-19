package b

import "example.com/renvotests/extended/multipackage/case059/pkg/a"

func Value() int {
	return 16 + a.Value() - a.Value()
}
