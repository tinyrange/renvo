#include <stdio.h>
#include <errno.h>
#include <string.h>
#define CHECK(x) do { if (!(x)) { printf("failed line %d errno=%d\n", __LINE__, errno); return 1; } } while (0)
int main(void) {
    FILE *f;
    FILE *all[FOPEN_MAX];
    char data[32];
    int i;
    errno = 0;
    CHECK(fopen("missing", "r") == NULL && errno == ENOENT);
    CHECK(fopen("data", "invalid") == NULL && errno == EINVAL);
    f = fopen("data", "wb");
    CHECK(f != NULL && !ferror(f) && !feof(f));
    CHECK(fwrite("hello\nworld", 1, 11, f) == 11);
    CHECK(fwrite("ignored", 0, 9, f) == 0);
    CHECK(fflush(f) == 0 && fflush(NULL) == 0);
    CHECK(fclose(f) == 0);
    f = fopen("data", "wx");
    CHECK(f == NULL && errno == EEXIST);
    f = fopen("data", "rb");
    CHECK(f != NULL && fileno(f) >= 0);
    CHECK(fgets(data, sizeof(data), f) == data && strcmp(data, "hello\n") == 0);
    CHECK(fread(data, 2, 4, f) == 2);
    CHECK(memcmp(data, "world", 5) == 0 && feof(f) && !ferror(f));
    CHECK(ungetc(255, f) == 255 && !feof(f));
    CHECK(getc(f) == 255 && getc(f) == EOF && feof(f));
    CHECK(fputc('!', f) == EOF && ferror(f) && errno == EBADF);
    clearerr(f);
    CHECK(!ferror(f) && !feof(f));
    CHECK(fclose(f) == 0);
    f = fopen("data", "a+");
    CHECK(f != NULL);
    CHECK(fgetc(f) == 'h');
    while (fgetc(f) != EOF) {}
    CHECK(feof(f) && !ferror(f));
    CHECK(fputs("!", f) >= 0 && fclose(f) == 0);
    f = fopen("data", "r+");
    CHECK(f != NULL && fputc('H', f) == 'H' && fclose(f) == 0);
    f = fopen("data", "r");
    CHECK(fread(data, 1, sizeof(data), f) == 12);
    CHECK(memcmp(data, "Hello\nworld!", 12) == 0 && fclose(f) == 0);
    f = fopen("data", "w");
    CHECK(f != NULL && fclose(f) == 0);
    f = fopen("data", "r");
    CHECK(f != NULL && fgetc(f) == EOF && feof(f) && fclose(f) == 0);
    for (i = 0; i < FOPEN_MAX * 2; i++) {
        f = fopen("data", "r"); CHECK(f != NULL && fclose(f) == 0);
    }
    for (i = 0; i < FOPEN_MAX - 3; i++) { all[i] = fopen("data", "r"); CHECK(all[i] != NULL); }
    CHECK(fopen("data", "r") == NULL && errno == EMFILE);
    for (i = 0; i < FOPEN_MAX - 3; i++) CHECK(fclose(all[i]) == 0);
    f = fopen("data", "r");
    CHECK(f != NULL && fread(data, 0, 1, f) == 0 && !feof(f));
    CHECK(fclose(f) == 0);
    puts("PASS");
    return 0;
}
