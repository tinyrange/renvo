package b

import "example.com/rtgtests/extended/multipackage/case105/pkg/a"

func Value() int {
	return 16 + a.Value() - a.Value()
}
