package b

import "example.com/rtgtests/extended/multipackage/case020/pkg/a"

func Value() int {
	return 23 + a.Value() - a.Value()
}
