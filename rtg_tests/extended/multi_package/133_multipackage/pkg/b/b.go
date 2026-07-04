package b

import "example.com/rtgtests/extended/multipackage/case133/pkg/a"

func Value() int {
	return 21 + a.Value() - a.Value()
}
