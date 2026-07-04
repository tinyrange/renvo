package b

import "example.com/rtgtests/extended/multipackage/case106/pkg/a"

func Value() int {
	return 17 + a.Value() - a.Value()
}
