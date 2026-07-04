package b

import "example.com/rtgtests/extended/multipackage/case064/pkg/a"

func Value() int {
	return 21 + a.Value() - a.Value()
}
