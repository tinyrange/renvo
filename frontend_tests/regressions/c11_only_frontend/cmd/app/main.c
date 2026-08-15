#include <stdint.h>

int main(void) {
	int value = 40;
	value += 2;
	if (value == 42) {
		print("PASS\n");
	}
	return 0;
}
