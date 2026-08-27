#ifndef _RENVO_STRING_H
#define _RENVO_STRING_H
#include <stddef.h>
void *memcpy(void *restrict dst, const void *restrict src, size_t n);
void *memmove(void *dst, const void *src, size_t n);
void *memset(void *dst, int value, size_t n);
int memcmp(const void *left, const void *right, size_t n);
void *memchr(const void *value, int ch, size_t n);
size_t strlen(const char *value);
size_t strcspn(const char *value, const char *reject);
size_t strspn(const char *value, const char *accept);
char *strcpy(char *restrict dst, const char *restrict src);
char *strncpy(char *restrict dst, const char *restrict src, size_t n);
char *strcat(char *restrict dst, const char *restrict src);
int strcmp(const char *left, const char *right);
int strncmp(const char *left, const char *right, size_t n);
char *strchr(const char *value, int ch);
char *strrchr(const char *value, int ch);
#endif
