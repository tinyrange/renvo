package b

import "example.com/rtgtests/extended/multipackage/case149/pkg/a"

func Value() int {
	return 14 + a.Value() - a.Value()
}
