#include <sys/resource.h>
#include <errno.h>
extern int __renvo_c_getrlimit(int resource, void *limit);
int getrlimit(int resource, struct rlimit *limit) {
 int result = __renvo_c_getrlimit(resource, limit);
 if (result < 0) { errno = -result; return -1; }
 return result;
}
