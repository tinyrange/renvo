/*
 * The application loop is ordinary C11. _cgo_export.h declares the two board
 * adapters explicitly exported by the Go file.
 */
#include "_cgo_export.h"

void cBlinkForever(int intervalMilliseconds) {
	for (;;) {
		goSetBlueLED(1);
		goDelayMilliseconds(intervalMilliseconds);
		goSetBlueLED(0);
		goDelayMilliseconds(intervalMilliseconds);
	}
}
