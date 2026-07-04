package b

import "example.com/rtgtests/extended/multipackage/case039/pkg/a"

func Value() int {
	return 19 + a.Value() - a.Value()
}
