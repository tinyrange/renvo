#ifndef _RENVO_WCHAR_H
#define _RENVO_WCHAR_H
#include <stddef.h>
#include <stdarg.h>
#include <stdio.h>
typedef unsigned int wint_t;
#define WEOF ((wint_t)-1)
#define WCHAR_MIN (-2147483647 - 1)
#define WCHAR_MAX 2147483647
typedef struct __renvo_mbstate { unsigned int value; unsigned int minimum; int remaining; } mbstate_t;
int mbsinit(const mbstate_t *state);
size_t mbrtowc(wchar_t *restrict out, const char *restrict text, size_t size, mbstate_t *restrict state);
size_t mbrlen(const char *restrict text, size_t size, mbstate_t *restrict state);
size_t wcrtomb(char *restrict out, wchar_t ch, mbstate_t *restrict state);
size_t mbsrtowcs(wchar_t *restrict out, const char **restrict text, size_t size, mbstate_t *restrict state);
size_t wcsrtombs(char *restrict out, const wchar_t **restrict text, size_t size, mbstate_t *restrict state);
size_t wcslen(const wchar_t *text);
int wcscmp(const wchar_t *left, const wchar_t *right);
wchar_t *wcscpy(wchar_t *restrict out, const wchar_t *restrict text);
wint_t fputwc(wchar_t ch, FILE *stream);
wint_t putwc(wchar_t ch, FILE *stream);
wint_t putwchar(wchar_t ch);
int fputws(const wchar_t *restrict text, FILE *restrict stream);
int vfwprintf(FILE *restrict stream, const wchar_t *restrict format, va_list args);
int fwprintf(FILE *restrict stream, const wchar_t *restrict format, ...);
int vwprintf(const wchar_t *restrict format, va_list args);
int wprintf(const wchar_t *restrict format, ...);
#endif
