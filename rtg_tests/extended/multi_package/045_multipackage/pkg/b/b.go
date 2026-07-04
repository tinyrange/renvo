package b

import "example.com/rtgtests/extended/multipackage/case045/pkg/a"

func Value() int {
	return 25 + a.Value() - a.Value()
}
