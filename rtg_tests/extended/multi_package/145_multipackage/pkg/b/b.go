package b

import "example.com/rtgtests/extended/multipackage/case145/pkg/a"

func Value() int {
	return 10 + a.Value() - a.Value()
}
