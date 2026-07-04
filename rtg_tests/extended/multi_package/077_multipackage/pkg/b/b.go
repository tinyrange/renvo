package b

import "example.com/rtgtests/extended/multipackage/case077/pkg/a"

func Value() int {
	return 11 + a.Value() - a.Value()
}
