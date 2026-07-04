package b

import "example.com/rtgtests/extended/multipackage/case110/pkg/a"

func Value() int {
	return 21 + a.Value() - a.Value()
}
