package b

import "example.com/rtgtests/extended/multipackage/case121/pkg/a"

func Value() int {
	return 9 + a.Value() - a.Value()
}
