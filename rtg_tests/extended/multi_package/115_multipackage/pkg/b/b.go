package b

import "example.com/rtgtests/extended/multipackage/case115/pkg/a"

func Value() int {
	return 3 + a.Value() - a.Value()
}
