package b

import "example.com/rtgtests/extended/multipackage/case012/pkg/a"

func Value() int {
	return 15 + a.Value() - a.Value()
}
