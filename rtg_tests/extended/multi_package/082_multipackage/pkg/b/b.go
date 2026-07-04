package b

import "example.com/rtgtests/extended/multipackage/case082/pkg/a"

func Value() int {
	return 16 + a.Value() - a.Value()
}
