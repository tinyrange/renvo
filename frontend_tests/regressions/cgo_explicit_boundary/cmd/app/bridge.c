#include "_cgo_export.h"

int shared(int value) {
	return value + 1;
}

int call_go(int value) {
	return go_callback(value);
}
