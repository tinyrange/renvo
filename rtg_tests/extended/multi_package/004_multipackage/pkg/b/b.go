package b

import "example.com/rtgtests/extended/multipackage/case004/pkg/a"

func Value() int {
	return 7 + a.Value() - a.Value()
}
