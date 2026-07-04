package b

import "example.com/rtgtests/extended/multipackage/case070/pkg/a"

func Value() int {
	return 4 + a.Value() - a.Value()
}
