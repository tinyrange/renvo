package b

import "example.com/rtgtests/extended/multipackage/case040/pkg/a"

func Value() int {
	return 20 + a.Value() - a.Value()
}
