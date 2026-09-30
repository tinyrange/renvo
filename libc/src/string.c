#include <string.h>
#if defined __linux__ && defined __x86_64__
#include <stdlib.h>
#endif

void *memchr(const void *value, int ch, size_t n) {
    const unsigned char *bytes = value;
    size_t i;
    for (i = 0; i < n; i++) {
        if (bytes[i] == (unsigned char)ch) return (void *)(bytes + i);
    }
    return NULL;
}

size_t strcspn(const char *value, const char *reject) {
    size_t count = 0;
    while (value[count] != '\0') {
        const char *candidate = reject;
        while (*candidate != '\0') {
            if (value[count] == *candidate) return count;
            candidate++;
        }
        count++;
    }
    return count;
}

size_t strspn(const char *value, const char *accept) {
    size_t count = 0;
    while (value[count] != '\0') {
        const char *candidate = accept;
        while (*candidate != '\0' && value[count] != *candidate) candidate++;
        if (*candidate == '\0') return count;
        count++;
    }
    return count;
}

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

char *strerror(int error) {
    switch (error) {
    case 0: return "Success";
    case 1: return "Operation not permitted";
    case 2: return "No such file or directory";
    case 4: return "Interrupted system call";
    case 5: return "Input/output error";
    case 9: return "Bad file descriptor";
    case 12: return "Cannot allocate memory";
    case 13: return "Permission denied";
    case 17: return "File exists";
    case 20: return "Not a directory";
    case 21: return "Is a directory";
    case 22: return "Invalid argument";
    case 24: return "Too many open files";
    case 28: return "No space left on device";
    case 29: return "Illegal seek";
    case 32: return "Broken pipe";
    case 34: return "Numerical result out of range";
    case 36: return "File name too long";
    case 38: return "Function not implemented";
    case 75: return "Value too large for defined data type";
    case 84: return "Invalid or incomplete multibyte or wide character";
    default: return "Unknown error";
    }
}

#if defined __linux__ && defined __x86_64__
char *strndup(const char *value, size_t n) {
    size_t len = 0;
    while (len < n && value[len] != 0) len++;
    char *copy = malloc(len + 1);
    if (copy == NULL) return NULL;
    memcpy(copy, value, len);
    copy[len] = 0;
    return copy;
}
char *strdup(const char *value) {
    return strndup(value, strlen(value));
}
#endif

char *strstr(const char *haystack, const char *needle) {
 if (!*needle) return (char *)haystack;
 for (; *haystack; haystack++) {
  size_t i=0;
  while (needle[i] && haystack[i] && needle[i]==haystack[i]) i++;
  if (!needle[i]) return (char *)haystack;
 }
 return NULL;
}
size_t strnlen(const char *text, size_t limit) { size_t i=0; while (i<limit && text[i]) i++; return i; }
