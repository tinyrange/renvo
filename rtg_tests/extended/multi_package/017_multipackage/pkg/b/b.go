package b

import "example.com/rtgtests/extended/multipackage/case017/pkg/a"

func Value() int {
	return 20 + a.Value() - a.Value()
}
