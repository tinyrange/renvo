package b

import "example.com/rtgtests/extended/multipackage/case147/pkg/a"

func Value() int {
	return 12 + a.Value() - a.Value()
}
