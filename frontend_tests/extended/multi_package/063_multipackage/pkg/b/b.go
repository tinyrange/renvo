package b

import "example.com/renvotests/extended/multipackage/case063/pkg/a"

func Value() int {
	return 20 + a.Value() - a.Value()
}
