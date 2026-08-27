extern int go_callback(int value);

int shared(int value) {
	return value + 1;
}

int call_go(int value) {
	return go_callback(value);
}
