#ifndef _RENVO_ERRNO_H
#define _RENVO_ERRNO_H
/* Single-threaded target runtime error state; numeric values follow Linux. */
extern int errno;
#if defined __linux__ && defined __x86_64__
extern char *program_invocation_name;
extern char *program_invocation_short_name;
#endif
#define EPERM 1
#define ENOENT 2
#define EINTR 4
#define EIO 5
#define EBADF 9
#define ENOMEM 12
#define EACCES 13
#define EFAULT 14
#define EEXIST 17
#define ENOTDIR 20
#define EISDIR 21
#define EINVAL 22
#define EMFILE 24
#define ENOSPC 28
#define ESPIPE 29
#define EPIPE 32
#define ERANGE 34
#define EILSEQ 84
#define ENAMETOOLONG 36
#define ENOSYS 38
#define EOVERFLOW 75
#endif
