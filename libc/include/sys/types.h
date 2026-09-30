#ifndef _RENVO_SYS_TYPES_H
#define _RENVO_SYS_TYPES_H

#include <stddef.h>

/* Linux amd64 ABI types, independent of the compiler host. */
#if !defined __linux__ || !defined __x86_64__
#error Renvo sys/types.h currently implements the Linux amd64 ABI only
#endif
typedef __PTRDIFF_TYPE__ ssize_t;
typedef int pid_t;
typedef unsigned int uid_t;
typedef unsigned int gid_t;
typedef unsigned int mode_t;
typedef unsigned long nlink_t;
typedef unsigned long long dev_t;
typedef unsigned long ino_t;
typedef long blksize_t;
#if __SIZEOF_LONG__ == 8
typedef long off_t;
typedef long blkcnt_t;
typedef long time_t;
#else
typedef long long off_t;
typedef long long blkcnt_t;
typedef long long time_t;
#endif

#endif
