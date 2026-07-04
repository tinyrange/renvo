package b

import "example.com/rtgtests/extended/multipackage/case143/pkg/a"

func Value() int {
	return 8 + a.Value() - a.Value()
}
