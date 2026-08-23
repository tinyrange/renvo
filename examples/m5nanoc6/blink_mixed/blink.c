/*
 * The application loop is ordinary C11. Renvo links these declarations to
 * package-level Go functions with the same names.
 */
extern void goSetBlueLED(int on);
extern void goDelayMilliseconds(int milliseconds);

void cBlinkForever(int intervalMilliseconds) {
	for (;;) {
		goSetBlueLED(1);
		goDelayMilliseconds(intervalMilliseconds);
		goSetBlueLED(0);
		goDelayMilliseconds(intervalMilliseconds);
	}
}
