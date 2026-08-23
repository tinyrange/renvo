#ifndef _RENVO_STDDEF_H
#define _RENVO_STDDEF_H

typedef __SIZE_TYPE__ size_t;
typedef __PTRDIFF_TYPE__ ptrdiff_t;
typedef int wchar_t;
#define NULL ((void *)0)
#define offsetof(type, member) __builtin_offsetof(type, member)

#endif
