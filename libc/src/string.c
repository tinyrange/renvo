#include <string.h>

void *memcpy(void *restrict dst, const void *restrict src, size_t n) {
    unsigned char *out = dst;
    const unsigned char *in = src;
    size_t i;
    for (i = 0; i < n; i++) out[i] = in[i];
    return dst;
}

void *memmove(void *dst, const void *src, size_t n) {
    unsigned char *out = dst;
    const unsigned char *in = src;
    size_t i;
    if (out < in) {
        for (i = 0; i < n; i++) out[i] = in[i];
    } else if (out > in) {
        for (i = n; i != 0; i--) out[i - 1] = in[i - 1];
    }
    return dst;
}

void *memset(void *dst, int value, size_t n) {
    unsigned char *out = dst;
    size_t i;
    for (i = 0; i < n; i++) out[i] = (unsigned char)value;
    return dst;
}

int memcmp(const void *left, const void *right, size_t n) {
    const unsigned char *a = left;
    const unsigned char *b = right;
    size_t i;
    for (i = 0; i < n; i++) {
        if (a[i] != b[i]) return a[i] < b[i] ? -1 : 1;
    }
    return 0;
}

size_t strlen(const char *value) {
    size_t n = 0;
    while (value[n] != '\0') n++;
    return n;
}

char *strcpy(char *restrict dst, const char *restrict src) {
    size_t i = 0;
    do { dst[i] = src[i]; } while (src[i++] != '\0');
    return dst;
}

char *strncpy(char *restrict dst, const char *restrict src, size_t n) {
    size_t i = 0;
    while (i < n && src[i] != '\0') { dst[i] = src[i]; i++; }
    while (i < n) dst[i++] = '\0';
    return dst;
}

char *strcat(char *restrict dst, const char *restrict src) {
    strcpy(dst + strlen(dst), src);
    return dst;
}

int strcmp(const char *left, const char *right) {
    while (*left != '\0' && *left == *right) { left++; right++; }
    return (unsigned char)*left - (unsigned char)*right;
}

int strncmp(const char *left, const char *right, size_t n) {
    size_t i;
    for (i = 0; i < n; i++) {
        unsigned char a = (unsigned char)left[i];
        unsigned char b = (unsigned char)right[i];
        if (a != b) return (int)a - (int)b;
        if (a == 0) return 0;
    }
    return 0;
}

char *strchr(const char *value, int ch) {
    char wanted = (char)ch;
    for (;;) {
        if (*value == wanted) return (char *)value;
        if (*value++ == '\0') return NULL;
    }
}

char *strrchr(const char *value, int ch) {
    const char *found = NULL;
    char wanted = (char)ch;
    do { if (*value == wanted) found = value; } while (*value++ != '\0');
    return (char *)found;
}
