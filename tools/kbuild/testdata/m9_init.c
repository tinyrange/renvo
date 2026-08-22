#include <sys/reboot.h>
#include <unistd.h>

int main(void) {
	static const char marker[] = "RENVO-LINUX-M9: PASS\n";
	(void)write(STDOUT_FILENO, marker, sizeof(marker) - 1);
	(void)reboot(RB_AUTOBOOT);
	for (;;) {
		(void)pause();
	}
}
