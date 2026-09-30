#ifndef _RENVO_STDIO_H
#define _RENVO_STDIO_H
#include <stddef.h>
#include <stdarg.h>
#define EOF (-1)
#define FOPEN_MAX 16
#define BUFSIZ 1024
#define _IOFBF 0
#define _IOLBF 1
#define _IONBF 2
typedef struct __renvo_FILE FILE;
extern FILE *stdin;
extern FILE *stdout;
extern FILE *stderr;
FILE *fopen(const char *restrict filename, const char *restrict mode);
int fclose(FILE *stream);
int fflush(FILE *stream);
int feof(FILE *stream);
int ferror(FILE *stream);
void clearerr(FILE *stream);
int fileno(FILE *stream);
int fgetc(FILE *stream);
int getc(FILE *stream);
int putc(int ch, FILE *stream);
int ungetc(int ch, FILE *stream);
char *fgets(char *restrict text, int size, FILE *restrict stream);
int putchar(int ch);
int getchar(void);
int puts(const char *text);
int fputc(int ch, FILE *stream);
int fputs(const char *restrict text, FILE *restrict stream);
size_t fwrite(const void *restrict ptr, size_t size, size_t count, FILE *restrict stream);
size_t fread(void *restrict ptr, size_t size, size_t count, FILE *restrict stream);
int sprintf(char *restrict buffer, const char *restrict format, ...);
int snprintf(char *restrict buffer, size_t size, const char *restrict format, ...);
int vsprintf(char *restrict buffer, const char *restrict format, va_list args);
int vsnprintf(char *restrict buffer, size_t size, const char *restrict format, va_list args);
int printf(const char *restrict format, ...);
int vprintf(const char *restrict format, va_list args);
int fprintf(FILE *restrict stream, const char *restrict format, ...);
int vfprintf(FILE *restrict stream, const char *restrict format, va_list args);
#endif
