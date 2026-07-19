package b

import "example.com/renvotests/extended/multipackage/case036/pkg/a"

func Value() int {
	return 16 + a.Value() - a.Value()
}
