package b

import "example.com/rtgtests/extended/multipackage/case006/pkg/a"

func Value() int {
	return 9 + a.Value() - a.Value()
}
