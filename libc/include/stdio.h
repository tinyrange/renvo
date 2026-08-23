#ifndef _RENVO_STDIO_H
#define _RENVO_STDIO_H
#include <stddef.h>
#include <stdarg.h>
#define EOF (-1)
typedef struct __renvo_FILE FILE;
extern FILE *stdin;
extern FILE *stdout;
extern FILE *stderr;
int putchar(int ch);
int getchar(void);
int puts(const char *text);
int fputc(int ch, FILE *stream);
int fputs(const char *restrict text, FILE *restrict stream);
size_t fwrite(const void *restrict ptr, size_t size, size_t count, FILE *restrict stream);
size_t fread(void *restrict ptr, size_t size, size_t count, FILE *restrict stream);
int printf(const char *restrict format, ...);
int vprintf(const char *restrict format, va_list args);
int fprintf(FILE *restrict stream, const char *restrict format, ...);
int vfprintf(FILE *restrict stream, const char *restrict format, va_list args);
#endif
