#ifndef _RENVO_STDLIB_H
#define _RENVO_STDLIB_H
#include <stddef.h>
#define EXIT_SUCCESS 0
#define EXIT_FAILURE 1
void *malloc(size_t size);
void *calloc(size_t count, size_t size);
void *realloc(void *ptr, size_t size);
void free(void *ptr);
int atoi(const char *text);
long atol(const char *text);
long long atoll(const char *text);
long strtol(const char *restrict text, char **restrict end, int base);
unsigned long strtoul(const char *restrict text, char **restrict end, int base);
int abs(int value);
long labs(long value);
void abort(void);
void exit(int status);
#endif
