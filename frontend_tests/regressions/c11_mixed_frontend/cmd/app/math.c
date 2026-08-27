#include <stdint.h>
#include "_cgo_export.h"

int cAdd(int left, int right) {
	int result = left + right;
	return result;
}

int cCallGo(int value) {
	return goDouble(value);
}
