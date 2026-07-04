package b

import "example.com/rtgtests/extended/multipackage/case100/pkg/a"

func Value() int {
	return 11 + a.Value() - a.Value()
}
