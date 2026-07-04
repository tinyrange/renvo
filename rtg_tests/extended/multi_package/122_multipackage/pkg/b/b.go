package b

import "example.com/rtgtests/extended/multipackage/case122/pkg/a"

func Value() int {
	return 10 + a.Value() - a.Value()
}
