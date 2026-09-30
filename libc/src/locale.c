#include <locale.h>
#include <stdlib.h>
#include <string.h>

/* Only the C/POSIX locale and its UTF-8 character encoding variant are installed.
   Unknown locale names fail without changing any category. */
static int __renvo_categories[6];
static const char *__renvo_category_names[6] = {
    "LC_CTYPE", "LC_NUMERIC", "LC_TIME", "LC_COLLATE", "LC_MONETARY", "LC_MESSAGES"
};
static char __renvo_locale_result[256];
static int __renvo_locale_code(const char *name) {
    if (!strcmp(name, "C") || !strcmp(name, "POSIX")) return 0;
    if (!strcmp(name, "C.UTF-8") || !strcmp(name, "C.utf8")) return 1;
    return -1;
}
static const char *__renvo_locale_name(int code) { return code ? "C.UTF-8" : "C"; }
char *setlocale(int category, const char *locale) {
    int i, start, end, codes[6], same;
    const char *name;
    if (category < 0 || category > LC_ALL) return NULL;
    start = category == LC_ALL ? 0 : category;
    end = category == LC_ALL ? 6 : category + 1;
    if (locale != NULL && category == LC_ALL && strncmp(locale, "LC_", 3) == 0) {
        const char *cursor = locale;
        char part[16];
        size_t n, used;
        for (i = 0; i < 6; i++) {
            n = strlen(__renvo_category_names[i]);
            if (strncmp(cursor, __renvo_category_names[i], n) || cursor[n] != '=') return NULL;
            cursor += n + 1;
            used = 0;
            while (*cursor && *cursor != ';' && used < sizeof(part)-1) part[used++] = *cursor++;
            part[used] = '\0';
            codes[i] = __renvo_locale_code(part);
            if (codes[i] < 0 || (i < 5 ? *cursor != ';' : *cursor != '\0')) return NULL;
            if (i < 5) cursor++;
        }
        for (i = 0; i < 6; i++) __renvo_categories[i] = codes[i];
    } else if (locale != NULL) {
        for (i = start; i < end; i++) {
            name = locale;
            if (*name == '\0') {
                name = getenv("LC_ALL");
                if (name == NULL || *name == '\0') name = getenv(__renvo_category_names[i]);
                if (name == NULL || *name == '\0') name = getenv("LANG");
                if (name == NULL || *name == '\0') name = "C";
            }
            codes[i] = __renvo_locale_code(name);
            if (codes[i] < 0) return NULL;
        }
        for (i = start; i < end; i++) __renvo_categories[i] = codes[i];
    }
    __renvo_mb_cur_max = __renvo_categories[LC_CTYPE] ? 4 : 1;
    same = 1;
    for (i = start + 1; i < end; i++) if (__renvo_categories[i] != __renvo_categories[start]) same = 0;
    if (same) return (char *)__renvo_locale_name(__renvo_categories[start]);
    __renvo_locale_result[0] = '\0';
    for (i = 0; i < 6; i++) {
        if (i) strcat(__renvo_locale_result, ";");
        strcat(__renvo_locale_result, __renvo_category_names[i]);
        strcat(__renvo_locale_result, "=");
        strcat(__renvo_locale_result, __renvo_locale_name(__renvo_categories[i]));
    }
    return __renvo_locale_result;
}
int __renvo_locale_utf8(void) { return __renvo_categories[LC_CTYPE]; }
