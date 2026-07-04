package b

import "example.com/rtgtests/extended/multipackage/case032/pkg/a"

func Value() int {
	return 12 + a.Value() - a.Value()
}
