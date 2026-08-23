#include <assert.h>
#include <stdio.h>
#include <stdlib.h>
void __renvo_assert_fail(const char *expression, const char *file, int line) {
    printf("assertion failed: %s (%s:%d)\n", expression, file, line);
    abort();
}
