package b

import "example.com/rtgtests/extended/multipackage/case036/pkg/a"

func Value() int {
	return 16 + a.Value() - a.Value()
}
