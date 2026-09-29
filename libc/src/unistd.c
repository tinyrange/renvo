#include <unistd.h>
extern void __renvo_c_abort(int status);
void _exit(int status) { __renvo_c_abort(status); for (;;) {} }

#if defined __linux__ && defined __x86_64__
#include <errno.h>
extern int __renvo_c_dup2(int, int);
int dup2(int oldfd, int newfd) {
    int result = __renvo_c_dup2(oldfd, newfd);
    if (result < 0) { errno = -result; return -1; }
    return result;
}
#endif

#include <errno.h>
extern int __renvo_c_close(int);
int close(int fd) {
    int result = __renvo_c_close(fd);
    if (result < 0) { errno = -result; return -1; }
    return result;
}
