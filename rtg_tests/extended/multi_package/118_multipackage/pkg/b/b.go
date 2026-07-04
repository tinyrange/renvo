package b

import "example.com/rtgtests/extended/multipackage/case118/pkg/a"

func Value() int {
	return 6 + a.Value() - a.Value()
}
