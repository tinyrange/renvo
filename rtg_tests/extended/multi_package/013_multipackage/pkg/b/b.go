package b

import "example.com/rtgtests/extended/multipackage/case013/pkg/a"

func Value() int {
	return 16 + a.Value() - a.Value()
}
