package b

import "example.com/rtgtests/extended/multipackage/case021/pkg/a"

func Value() int {
	return 24 + a.Value() - a.Value()
}
