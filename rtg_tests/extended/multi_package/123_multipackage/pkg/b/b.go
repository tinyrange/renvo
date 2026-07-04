package b

import "example.com/rtgtests/extended/multipackage/case123/pkg/a"

func Value() int {
	return 11 + a.Value() - a.Value()
}
