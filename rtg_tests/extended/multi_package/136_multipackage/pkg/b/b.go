package b

import "example.com/rtgtests/extended/multipackage/case136/pkg/a"

func Value() int {
	return 24 + a.Value() - a.Value()
}
