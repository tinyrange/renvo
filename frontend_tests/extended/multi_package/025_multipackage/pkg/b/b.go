package b

import "example.com/renvotests/extended/multipackage/case025/pkg/a"

func Value() int {
	return 5 + a.Value() - a.Value()
}
