package b

import "example.com/rtgtests/extended/multipackage/case079/pkg/a"

func Value() int {
	return 13 + a.Value() - a.Value()
}
