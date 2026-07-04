package b

import "example.com/rtgtests/extended/multipackage/case134/pkg/a"

func Value() int {
	return 22 + a.Value() - a.Value()
}
