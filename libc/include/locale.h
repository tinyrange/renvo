#ifndef _RENVO_LOCALE_H
#define _RENVO_LOCALE_H
#include <stddef.h>
#define LC_CTYPE 0
#define LC_NUMERIC 1
#define LC_TIME 2
#define LC_COLLATE 3
#define LC_MONETARY 4
#define LC_MESSAGES 5
#define LC_ALL 6
char *setlocale(int category, const char *locale);
/* Internal encoding selection, shared by restartable conversion functions. */
int __renvo_locale_utf8(void);
#endif
