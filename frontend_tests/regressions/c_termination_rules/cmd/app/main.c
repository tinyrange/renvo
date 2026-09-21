int forever(void) {
	for (;;) {}
}

int another_forever(void) {
	while (1) {}
}

int main(void) {
	print("PASS\n");
	return 0;
}
