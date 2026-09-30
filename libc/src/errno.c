#include <errno.h>
#if defined __linux__ && defined __x86_64__
/* Program-name globals require the hosted process initialization/exit wrapper. */
#include <stdlib.h>
#endif
int errno;
