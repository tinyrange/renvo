package b

import "example.com/rtgtests/extended/multipackage/case054/pkg/a"

func Value() int {
	return 11 + a.Value() - a.Value()
}
