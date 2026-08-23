#include <stdlib.h>
#include <string.h>
#include <stdint.h>

#if defined(__STDC_HOSTED__) && __STDC_HOSTED__
static unsigned char __renvo_heap[1024 * 1024];
static size_t __renvo_heap_used;

void *malloc(size_t size) {
    size_t align = sizeof(uintptr_t);
    size_t header = 2 * sizeof(size_t);
    size_t start = (__renvo_heap_used + align - 1) & ~(align - 1);
    size_t total;
    size_t *meta;
    if (size == 0) size = 1;
    if (size > sizeof(__renvo_heap) - header || start > sizeof(__renvo_heap) - header - size) return NULL;
    total = header + size;
    meta = (size_t *)(__renvo_heap + start);
    meta[0] = size;
    meta[1] = 0x524e564fUL;
    __renvo_heap_used = start + total;
    return (void *)(meta + 2);
}

void *calloc(size_t count, size_t size) {
    void *result;
    if (size != 0 && count > (size_t)-1 / size) return NULL;
    result = malloc(count * size);
    if (result != NULL) memset(result, 0, count * size);
    return result;
}

void *realloc(void *ptr, size_t size) {
    void *result;
    size_t old_size;
    if (ptr == NULL) return malloc(size);
    if (size == 0) return NULL;
    old_size = ((size_t *)ptr)[-2];
    if (size <= old_size) return ptr;
    result = malloc(size);
    if (result != NULL) memcpy(result, ptr, old_size);
    return result;
}

void free(void *ptr) { (void)ptr; }
#else
#error "Renvo libc allocator has no implementation for this platform"
#endif

static int __renvo_digit(int ch) {
    if (ch >= '0' && ch <= '9') return ch - '0';
    if (ch >= 'a' && ch <= 'z') return ch - 'a' + 10;
    if (ch >= 'A' && ch <= 'Z') return ch - 'A' + 10;
    return -1;
}

unsigned long strtoul(const char *restrict text, char **restrict end, int base) {
    const char *start = text;
    unsigned long value = 0;
    int digit;
    while (*text == ' ' || (*text >= '\t' && *text <= '\r')) text++;
    if (*text == '+') text++;
    if ((base == 0 || base == 16) && text[0] == '0' && (text[1] == 'x' || text[1] == 'X')) { base = 16; text += 2; }
    else if (base == 0 && *text == '0') base = 8;
    else if (base == 0) base = 10;
    while ((digit = __renvo_digit(*text)) >= 0 && digit < base) { value = value * (unsigned)base + (unsigned)digit; text++; }
    if (end != NULL) *end = (char *)(text == start ? start : text);
    return value;
}

long strtol(const char *restrict text, char **restrict end, int base) {
    int negative = 0;
    while (*text == ' ' || (*text >= '\t' && *text <= '\r')) text++;
    if (*text == '-') { negative = 1; text++; }
    else if (*text == '+') text++;
    {
        unsigned long value = strtoul(text, end, base);
        return negative ? -(long)value : (long)value;
    }
}

int atoi(const char *text) { return (int)strtol(text, NULL, 10); }
long atol(const char *text) { return strtol(text, NULL, 10); }
long long atoll(const char *text) { return (long long)strtol(text, NULL, 10); }
int abs(int value) { return value < 0 ? -value : value; }
long labs(long value) { return value < 0 ? -value : value; }

extern void __renvo_c_abort(int status);
void abort(void) { __renvo_c_abort(EXIT_FAILURE); for (;;) {} }
void exit(int status) { __renvo_c_abort(status); for (;;) {} }
