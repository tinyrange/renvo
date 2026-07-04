package b

import "example.com/rtgtests/extended/multipackage/case060/pkg/a"

func Value() int {
	return 17 + a.Value() - a.Value()
}
