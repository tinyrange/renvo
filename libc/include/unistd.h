#ifndef _RENVO_UNISTD_H
#define _RENVO_UNISTD_H
#include <stddef.h>
#define STDIN_FILENO 0
#define STDOUT_FILENO 1
#define STDERR_FILENO 2
void _exit(int status);
int close(int fd);
#if defined __linux__ && defined __x86_64__
int dup2(int oldfd, int newfd);
#endif
#endif
