package b

import "example.com/rtgtests/extended/multipackage/case030/pkg/a"

func Value() int {
	return 10 + a.Value() - a.Value()
}
