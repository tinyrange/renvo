#ifndef _RENVO_SYS_RESOURCE_H
#define _RENVO_SYS_RESOURCE_H
#include <sys/types.h>
typedef unsigned long rlim_t;
struct rlimit { rlim_t rlim_cur; rlim_t rlim_max; };
#define RLIMIT_NOFILE 7
#define RLIM_INFINITY (~(rlim_t)0)
int getrlimit(int resource, struct rlimit *limit);
#endif
