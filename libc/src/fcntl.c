#include <fcntl.h>
#include <stdarg.h>
#include <errno.h>

extern int __renvo_c_fcntl(int fd, int action, int argument);
int fcntl(int fd, int action, ...) {
    int argument = 0;
    va_list args;
    switch (action) {
    case F_DUPFD: case F_DUPFD_CLOEXEC: case F_SETFD: case F_SETFL:
        va_start(args, action);
        argument = va_arg(args, int);
        va_end(args);
        break;
    case F_GETFD: case F_GETFL:
        break;
    default:
        errno = EINVAL;
        return -1;
    }
    int result = __renvo_c_fcntl(fd, action, argument);
    if (result < 0) { errno = -result; return -1; }
    return result;
}
