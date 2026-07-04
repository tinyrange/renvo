package b

import "example.com/rtgtests/extended/multipackage/case043/pkg/a"

func Value() int {
	return 23 + a.Value() - a.Value()
}
