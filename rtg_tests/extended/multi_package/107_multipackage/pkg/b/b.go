package b

import "example.com/rtgtests/extended/multipackage/case107/pkg/a"

func Value() int {
	return 18 + a.Value() - a.Value()
}
