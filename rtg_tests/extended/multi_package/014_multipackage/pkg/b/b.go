package b

import "example.com/rtgtests/extended/multipackage/case014/pkg/a"

func Value() int {
	return 17 + a.Value() - a.Value()
}
