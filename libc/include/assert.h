#ifndef _RENVO_ASSERT_H
#define _RENVO_ASSERT_H
#ifdef NDEBUG
#define assert(condition) ((void)0)
#else
void __renvo_assert_fail(const char *expression, const char *file, int line);
#define assert(condition) ((condition) ? (void)0 : __renvo_assert_fail(#condition, __FILE__, __LINE__))
#endif
#endif
