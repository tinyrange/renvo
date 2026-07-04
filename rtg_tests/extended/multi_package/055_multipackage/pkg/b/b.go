package b

import "example.com/rtgtests/extended/multipackage/case055/pkg/a"

func Value() int {
	return 12 + a.Value() - a.Value()
}
