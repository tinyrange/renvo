#include "sqlite3.h"
#include <stdio.h>

static int print_row(void *unused, int columns, char **values, char **names) {
    int i;
    (void)unused;
    for (i = 0; i < columns; i++) {
        printf("%s=%s%s", names[i], values[i] ? values[i] : "NULL",
            i + 1 == columns ? "\n" : " ");
    }
    return 0;
}

static int run(sqlite3 *database, const char *phase, const char *sql,
               sqlite3_callback callback) {
    char *error = 0;
    int status = sqlite3_exec(database, sql, callback, 0, &error);
    if (status != SQLITE_OK) {
        fprintf(stderr, "%s: %s\n", phase,
            error ? error : sqlite3_errmsg(database));
        sqlite3_free(error);
        return 0;
    }
    return 1;
}

int main(void) {
    sqlite3 *database = 0;
    int status = sqlite3_open(":memory:", &database);
    if (status != SQLITE_OK) {
        fprintf(stderr, "open: %s\n", sqlite3_errmsg(database));
        return 1;
    }
    if (!run(database, "query C",
            "select 'C' as name, 1972 as year", print_row) ||
        !run(database, "query SQL",
            "select 'SQL' as name, 1974 as year", print_row)) {
        sqlite3_close(database);
        return 2;
    }
    sqlite3_close(database);
    return 0;
}
