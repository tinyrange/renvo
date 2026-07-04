package b

import "example.com/rtgtests/extended/multipackage/case140/pkg/a"

func Value() int {
	return 5 + a.Value() - a.Value()
}
