package b

import "example.com/rtgtests/extended/multipackage/case112/pkg/a"

func Value() int {
	return 23 + a.Value() - a.Value()
}
