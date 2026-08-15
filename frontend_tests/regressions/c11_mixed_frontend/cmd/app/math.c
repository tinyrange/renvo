#include <stdint.h>

extern int goDouble(int value);

int cAdd(int left, int right) {
	int result = left + right;
	return result;
}

int cCallGo(int value) {
	return goDouble(value);
}
