#include <stdio.h>
#include <stdint.h>
#include <errno.h>

/* Unbuffered streams. A reusable pool avoids leaking stream allocations in
   the current bump-allocated target heap. Slots are released even on close error. */
enum { STREAM_USED = 1, STREAM_READ = 2, STREAM_WRITE = 4, STREAM_ERROR = 8, STREAM_EOF = 16 };
struct __renvo_FILE { int fd; unsigned flags; int pushed; };
static FILE __renvo_stdin = { 0, STREAM_USED | STREAM_READ, -1 };
static FILE __renvo_stdout = { 1, STREAM_USED | STREAM_WRITE, -1 };
static FILE __renvo_stderr = { 2, STREAM_USED | STREAM_WRITE, -1 };
FILE *stdin = &__renvo_stdin;
FILE *stdout = &__renvo_stdout;
FILE *stderr = &__renvo_stderr;

extern int __renvo_c_open(const char *name, int flags);
extern int __renvo_c_close(int fd);
extern int __renvo_c_write_byte(int fd, int ch);
extern int __renvo_c_read_byte(int fd, unsigned char *out);

FILE *fopen(const char *restrict filename, const char *restrict mode) {
    static FILE __renvo_streams[FOPEN_MAX - 3];
    int flags, plus = 0, binary = 0, exclusive = 0, fd, i;
    FILE *stream = NULL;
    char kind;
    if (filename == NULL || mode == NULL || *mode == '\0') { errno = EINVAL; return NULL; }
    kind = *mode++;
    if (kind != 'r' && kind != 'w' && kind != 'a') { errno = EINVAL; return NULL; }
    while (*mode) {
        if (*mode == '+' && !plus) plus = 1;
        else if (*mode == 'b' && !binary) binary = 1;
        else if (*mode == 'x' && !exclusive && kind == 'w') exclusive = 1;
        else { errno = EINVAL; return NULL; }
        mode++;
    }
    for (i = 0; i < FOPEN_MAX - 3; i++) {
        if (!(__renvo_streams[i].flags & STREAM_USED)) { stream = &__renvo_streams[i]; break; }
    }
    if (stream == NULL) { errno = EMFILE; return NULL; }
    /* Portable bridge flags: read=1, write=2, create=4, truncate=8,
       append=16, exclusive=32; translated to target flags by the bridge. */
    flags = kind == 'r' ? 1 : 2 | 4;
    if (plus) flags |= 3;
    if (kind == 'w') flags |= 8;
    if (kind == 'a') flags |= 16;
    if (exclusive) flags |= 32;
    fd = __renvo_c_open(filename, flags);
    if (fd < 0) { errno = -fd; return NULL; }
    stream->fd = fd;
    stream->flags = STREAM_USED;
    if (flags & 1) stream->flags |= STREAM_READ;
    if (flags & 2) stream->flags |= STREAM_WRITE;
    stream->pushed = -1;
    return stream;
}

int fclose(FILE *stream) {
    int result;
    if (stream == NULL || !(stream->flags & STREAM_USED)) { errno = EBADF; return EOF; }
    result = __renvo_c_close(stream->fd);
    stream->flags &= ~STREAM_USED;
    if (result < 0) { errno = -result; return EOF; }
    return 0;
}
int fflush(FILE *stream) {
    /* No pending output: every write has already reached the descriptor. */
    if (stream != NULL && !(stream->flags & STREAM_USED)) { errno = EBADF; return EOF; }
    return 0;
}
/* All output is unbuffered, so there are never pending bytes to flush. */
size_t __fpending(FILE *stream) { (void)stream; return 0; }
void __renvo_stream_error(FILE *stream, int code) { stream->flags |= STREAM_ERROR; errno = code; }
int feof(FILE *stream) { return (stream->flags & STREAM_EOF); }
int ferror(FILE *stream) { return (stream->flags & STREAM_ERROR); }
void clearerr(FILE *stream) { stream->flags &= ~STREAM_ERROR; stream->flags &= ~STREAM_EOF; }
int fileno(FILE *stream) {
    if (!(stream->flags & STREAM_USED)) { errno = EBADF; return -1; }
    return stream->fd;
}
int fputc(int ch, FILE *stream) {
    int result;
    if ((stream->flags & (STREAM_USED | STREAM_WRITE)) != (STREAM_USED | STREAM_WRITE)) { stream->flags |= STREAM_ERROR; errno = EBADF; return EOF; }
    result = __renvo_c_write_byte(stream->fd, (unsigned char)ch);
    if (result < 0) { stream->flags |= STREAM_ERROR; errno = -result; return EOF; }
    return (unsigned char)ch;
}
int fgetc(FILE *stream) {
    unsigned char ch;
    int result;
    if ((stream->flags & (STREAM_USED | STREAM_READ)) != (STREAM_USED | STREAM_READ)) { stream->flags |= STREAM_ERROR; errno = EBADF; return EOF; }
    if (stream->pushed != -1) { result = stream->pushed; stream->pushed = -1; return result; }
    if ((stream->flags & STREAM_EOF)) return EOF;
    result = __renvo_c_read_byte(stream->fd, &ch);
    if (result == 0) { stream->flags |= STREAM_EOF; return EOF; }
    if (result < 0) { stream->flags |= STREAM_ERROR; errno = -result; return EOF; }
    return ch;
}
int ungetc(int ch, FILE *stream) {
    if (ch == EOF || (stream->flags & (STREAM_USED | STREAM_READ)) != (STREAM_USED | STREAM_READ) || stream->pushed != -1) return EOF;
    stream->pushed = (unsigned char)ch;
    stream->flags &= ~STREAM_EOF;
    return stream->pushed;
}
int getc(FILE *stream) { return fgetc(stream); }
int putc(int ch, FILE *stream) { return fputc(ch, stream); }
int putchar(int ch) { return fputc(ch, stdout); }
int getchar(void) { return fgetc(stdin); }

char *fgets(char *restrict text, int size, FILE *restrict stream) {
    int i = 0, ch;
    if (size <= 0) return NULL;
    while (i < size - 1) {
        ch = fgetc(stream);
        if (ch == EOF) { if (ferror(stream) || i == 0) return NULL; break; }
        text[i++] = (char)ch;
        if (ch == '\n') break;
    }
    text[i] = '\0';
    return text;
}
int fputs(const char *restrict text, FILE *restrict stream) {
    int count = 0;
    while (*text != '\0') { if (fputc((unsigned char)*text++, stream) == EOF) return EOF; count++; }
    return count;
}
size_t fwrite(const void *restrict ptr, size_t size, size_t count, FILE *restrict stream) {
    const unsigned char *bytes = ptr;
    size_t total, i;
    if (size == 0 || count == 0) return 0;
    if (count > (size_t)-1 / size) { stream->flags |= STREAM_ERROR; errno = EOVERFLOW; return 0; }
    total = size * count;
    for (i = 0; i < total; i++) if (fputc(bytes[i], stream) == EOF) return i / size;
    return count;
}
size_t fread(void *restrict ptr, size_t size, size_t count, FILE *restrict stream) {
    unsigned char *bytes = ptr;
    size_t total, i;
    if (size == 0 || count == 0) return 0;
    if (count > (size_t)-1 / size) { stream->flags |= STREAM_ERROR; errno = EOVERFLOW; return 0; }
    total = size * count;
    for (i = 0; i < total; i++) {
        int ch = fgetc(stream);
        if (ch == EOF) return i / size;
        bytes[i] = (unsigned char)ch;
    }
    return count;
}
int puts(const char *text) {
    int count = fputs(text, stdout);
    if (count == EOF || putchar('\n') == EOF) return EOF;
    return count + 1;
}

struct __renvo_format_sink {
    FILE *file;
    char *buffer;
    size_t capacity, used;
};
static int __renvo_format_char(int ch, struct __renvo_format_sink *sink) {
    if (sink->file != NULL) return fputc(ch, sink->file);
    if (sink->capacity != 0 && sink->used < sink->capacity - 1)
        sink->buffer[sink->used] = (char)ch;
    sink->used++;
    return (unsigned char)ch;
}
static int __renvo_format_text(const char *text, struct __renvo_format_sink *sink) {
    int count = 0;
    while (*text) { if (__renvo_format_char((unsigned char)*text++, sink) == EOF) return EOF; count++; }
    return count;
}
static int __renvo_print_unsigned(struct __renvo_format_sink *stream, unsigned long long value, unsigned base, int upper, int width, int zero) {
    char digits[32];
    int used = 0;
    int count = 0;
    const char *alphabet = upper ? "0123456789ABCDEF" : "0123456789abcdef";
    do { digits[used++] = alphabet[value % base]; value /= base; } while (value != 0);
    while (width-- > used) { __renvo_format_char(zero ? '0' : ' ', stream); count++; }
    while (used != 0) { __renvo_format_char(digits[--used], stream); count++; }
    return count;
}

#if defined(__STDC_NO_IEC_60559_BFP__)
static int __renvo_print_float(struct __renvo_format_sink *stream, double value, int precision) {
    (void)value;
    (void)precision;
    return __renvo_format_text("<float unavailable>", stream);
}
#else
static int __renvo_print_float(struct __renvo_format_sink *stream, double value, int precision) {
    unsigned long long whole;
    double fraction;
    int count = 0;
    int i;
    if (value != value) { __renvo_format_text("nan", stream); return 3; }
    if (value < 0) { __renvo_format_char('-', stream); count++; value = -value; }
    whole = (unsigned long long)value;
    fraction = value - (double)whole;
    count += __renvo_print_unsigned(stream, whole, 10, 0, 0, 0);
    if (precision > 0) {
        __renvo_format_char('.', stream); count++;
        for (i = 0; i < precision; i++) {
            int digit;
            fraction *= 10.0;
            digit = (int)fraction;
            __renvo_format_char('0' + digit, stream);
            count++;
            fraction -= digit;
        }
    }
    return count;
}
#endif

static int __renvo_format(struct __renvo_format_sink *stream, const char *restrict format, va_list args) {
    int count = 0;
    while (*format != '\0') {
        int width = 0;
        int zero = 0;
        int long_count = 0;
        int precision = -1;
        char spec;
        if (*format != '%') { __renvo_format_char((unsigned char)*format++, stream); count++; continue; }
        format++;
        if (*format == '%') { __renvo_format_char('%', stream); format++; count++; continue; }
        if (*format == '0') { zero = 1; format++; }
        while (*format >= '0' && *format <= '9') { width = width * 10 + *format++ - '0'; }
        if (*format == '.') {
            precision = 0;
            format++;
            while (*format >= '0' && *format <= '9') { precision = precision * 10 + *format++ - '0'; }
        }
        while (*format == 'l') { long_count++; format++; }
        spec = *format++;
        if (spec == 's') {
            const char *text = va_arg(args, const char *);
            if (text == NULL) text = "(null)";
            while (*text != '\0') { __renvo_format_char((unsigned char)*text++, stream); count++; }
        } else if (spec == 'c') {
            __renvo_format_char(va_arg(args, int), stream); count++;
        } else if (spec == 'd' || spec == 'i') {
            long long value = long_count > 1 ? va_arg(args, long long) : long_count ? va_arg(args, long) : va_arg(args, int);
            unsigned long long magnitude;
            if (value < 0) { __renvo_format_char('-', stream); count++; magnitude = (unsigned long long)(-(value + 1)) + 1; }
            else magnitude = (unsigned long long)value;
            count += __renvo_print_unsigned(stream, magnitude, 10, 0, width, zero);
        } else if (spec == 'u' || spec == 'x' || spec == 'X' || spec == 'o') {
            unsigned long long value = long_count > 1 ? va_arg(args, unsigned long long) : long_count ? va_arg(args, unsigned long) : va_arg(args, unsigned int);
            unsigned base = spec == 'o' ? 8 : (spec == 'u' ? 10 : 16);
            count += __renvo_print_unsigned(stream, value, base, spec == 'X', width, zero);
        } else if (spec == 'p') {
            uintptr_t value = (uintptr_t)va_arg(args, void *);
            __renvo_format_char('0', stream); __renvo_format_char('x', stream); count += 2 + __renvo_print_unsigned(stream, value, 16, 0, (int)(2 * sizeof(void *)), 1);
        } else if (spec == 'f' || spec == 'F') {
            if (precision < 0) precision = 6;
            count += __renvo_print_float(stream, va_arg(args, double), precision);
        } else {
            __renvo_format_char('%', stream); __renvo_format_char(spec, stream); count += 2;
        }
    }
    return stream->file != NULL && ferror(stream->file) ? EOF : count;
}

int vfprintf(FILE *restrict stream, const char *restrict format, va_list args) {
    struct __renvo_format_sink sink = { stream, NULL, 0, 0 };
    return __renvo_format(&sink, format, args);
}
int vsnprintf(char *restrict buffer, size_t size, const char *restrict format, va_list args) {
    struct __renvo_format_sink sink = { NULL, buffer, size, 0 };
    int result = __renvo_format(&sink, format, args);
    if (size != 0) buffer[sink.used < size ? sink.used : size - 1] = 0;
    return result;
}
int vsprintf(char *restrict buffer, const char *restrict format, va_list args) {
    return vsnprintf(buffer, (size_t)-1, format, args);
}
int snprintf(char *restrict buffer, size_t size, const char *restrict format, ...) {
    int result;
    va_list args;
    va_start(args, format);
    result = vsnprintf(buffer, size, format, args);
    va_end(args);
    return result;
}
int sprintf(char *restrict buffer, const char *restrict format, ...) {
    int result;
    va_list args;
    va_start(args, format);
    result = vsprintf(buffer, format, args);
    va_end(args);
    return result;
}

int vprintf(const char *restrict format, va_list args) { return vfprintf(stdout, format, args); }

int printf(const char *restrict format, ...) {
    int result;
    va_list args;
    va_start(args, format);
    result = vprintf(format, args);
    va_end(args);
    return result;
}

int fprintf(FILE *restrict stream, const char *restrict format, ...) {
    int result;
    va_list args;
    va_start(args, format);
    result = vfprintf(stream, format, args);
    va_end(args);
    return result;
}
