package b

import "example.com/renvotests/extended/multipackage/case058/pkg/a"

func Value() int {
	return 15 + a.Value() - a.Value()
}
