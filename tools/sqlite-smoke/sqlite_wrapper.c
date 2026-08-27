#include "sqlite3.c"

/* The smoke build omits SQLite's date functions, but a few declarations
 * remain reachable in the amalgamation. Keep their deterministic hosted
 * stubs here instead of expanding Renvo libc solely for this local test. */
time_t time(time_t *timer) {
    if (timer != 0) *timer = 0;
    return 0;
}

struct tm *gmtime(const time_t *timer) {
    (void)timer;
    return 0;
}

struct tm *localtime(const time_t *timer) {
    (void)timer;
    return 0;
}

unsigned long strftime(char *value, unsigned long size, const char *format,
                       const struct tm *time_value) {
    (void)format;
    (void)time_value;
    if (size > 0) value[0] = 0;
    return 0;
}

/* SQLITE_OS_OTHER deliberately leaves storage to the embedding program.
 * This small growing-memory VFS is enough to exercise SQLite's pager and
 * callback APIs without adding host filesystem syscalls to the Renvo runtime. */
typedef struct RenvoMemoryFile RenvoMemoryFile;
struct RenvoMemoryFile {
    sqlite3_file base;
    unsigned char *data;
    sqlite3_int64 size;
    int capacity;
};

static int memoryClose(sqlite3_file *file) {
    RenvoMemoryFile *memory = (RenvoMemoryFile *)file;
    free(memory->data);
    memory->data = 0;
    memory->size = 0;
    memory->capacity = 0;
    return SQLITE_OK;
}

static int memoryRead(sqlite3_file *file, void *buffer, int amount,
                      sqlite3_int64 offset) {
    RenvoMemoryFile *memory = (RenvoMemoryFile *)file;
    int available = 0;
    if (offset < memory->size) {
        available = (int)(memory->size - offset);
        if (available > amount) available = amount;
        memcpy(buffer, memory->data + offset, (size_t)available);
    }
    if (available < amount) {
        memset((unsigned char *)buffer + available, 0,
            (size_t)(amount - available));
        return SQLITE_IOERR_SHORT_READ;
    }
    return SQLITE_OK;
}

static int memoryWrite(sqlite3_file *file, const void *buffer, int amount,
                       sqlite3_int64 offset) {
    RenvoMemoryFile *memory = (RenvoMemoryFile *)file;
    sqlite3_int64 end = offset + amount;
    if (end > memory->capacity) {
        int capacity = memory->capacity == 0 ? 4096 : memory->capacity;
        unsigned char *next;
        while (capacity < end) capacity *= 2;
        next = (unsigned char *)realloc(memory->data, (size_t)capacity);
        if (next == 0) return SQLITE_NOMEM;
        memory->data = next;
        memory->capacity = capacity;
    }
    memcpy(memory->data + offset, buffer, (size_t)amount);
    if (end > memory->size) memory->size = end;
    return SQLITE_OK;
}

static int memoryTruncate(sqlite3_file *file, sqlite3_int64 size) {
    RenvoMemoryFile *memory = (RenvoMemoryFile *)file;
    if (size < memory->size) memory->size = size;
    return SQLITE_OK;
}

static int memorySync(sqlite3_file *file, int flags) {
    (void)file;
    (void)flags;
    return SQLITE_OK;
}

static int memoryFileSize(sqlite3_file *file, sqlite3_int64 *size) {
    *size = ((RenvoMemoryFile *)file)->size;
    return SQLITE_OK;
}

static int memoryLock(sqlite3_file *file, int lock) {
    (void)file;
    (void)lock;
    return SQLITE_OK;
}

static int memoryUnlock(sqlite3_file *file, int lock) {
    (void)file;
    (void)lock;
    return SQLITE_OK;
}

static int memoryCheckReserved(sqlite3_file *file, int *result) {
    (void)file;
    *result = 0;
    return SQLITE_OK;
}

static int memoryFileControl(sqlite3_file *file, int operation,
                             void *argument) {
    (void)file;
    (void)operation;
    (void)argument;
    return SQLITE_NOTFOUND;
}

static int memorySectorSize(sqlite3_file *file) {
    (void)file;
    return 512;
}

static int memoryDeviceCharacteristics(sqlite3_file *file) {
    (void)file;
    return 0;
}

static const sqlite3_io_methods memoryMethods = {
    1, memoryClose, memoryRead, memoryWrite, memoryTruncate, memorySync,
    memoryFileSize, memoryLock, memoryUnlock, memoryCheckReserved,
    memoryFileControl, memorySectorSize, memoryDeviceCharacteristics
};

static int memoryOpen(sqlite3_vfs *vfs, const char *name, sqlite3_file *file,
                      int flags, int *out_flags) {
    RenvoMemoryFile *memory = (RenvoMemoryFile *)file;
    (void)vfs;
    (void)name;
    memset(memory, 0, sizeof(*memory));
    memory->base.pMethods = &memoryMethods;
    if (out_flags != 0) *out_flags = flags;
    return SQLITE_OK;
}

static int memoryDelete(sqlite3_vfs *vfs, const char *name, int sync_dir) {
    (void)vfs;
    (void)name;
    (void)sync_dir;
    return SQLITE_OK;
}

static int memoryAccess(sqlite3_vfs *vfs, const char *name, int flags,
                        int *result) {
    (void)vfs;
    (void)name;
    (void)flags;
    *result = 0;
    return SQLITE_OK;
}

static int memoryFullPath(sqlite3_vfs *vfs, const char *name, int size,
                          char *output) {
    (void)vfs;
    if (size <= 0) return SQLITE_CANTOPEN;
    strncpy(output, name == 0 ? "" : name, (size_t)size - 1);
    output[size - 1] = 0;
    return SQLITE_OK;
}

static void *memoryDlOpen(sqlite3_vfs *vfs, const char *name) {
    (void)vfs;
    (void)name;
    return 0;
}

static void memoryDlError(sqlite3_vfs *vfs, int size, char *message) {
    (void)vfs;
    if (size > 0) message[0] = 0;
}

static void (*memoryDlSym(sqlite3_vfs *vfs, void *handle,
                         const char *name))(void) {
    (void)vfs;
    (void)handle;
    (void)name;
    return 0;
}

static void memoryDlClose(sqlite3_vfs *vfs, void *handle) {
    (void)vfs;
    (void)handle;
}

static int memoryRandomness(sqlite3_vfs *vfs, int size, char *output) {
    (void)vfs;
    memset(output, 0x5a, (size_t)size);
    return size;
}

static int memorySleep(sqlite3_vfs *vfs, int microseconds) {
    (void)vfs;
    return microseconds;
}

static int memoryCurrentTime(sqlite3_vfs *vfs, double *value) {
    (void)vfs;
    *value = 2440587.5;
    return SQLITE_OK;
}

static int memoryGetLastError(sqlite3_vfs *vfs, int size, char *message) {
    (void)vfs;
    if (size > 0) message[0] = 0;
    return 0;
}

static sqlite3_vfs memoryVFS = {
    1, sizeof(RenvoMemoryFile), 256, 0, "renvo-memory", 0,
    memoryOpen, memoryDelete, memoryAccess, memoryFullPath,
    memoryDlOpen, memoryDlError, memoryDlSym, memoryDlClose,
    memoryRandomness, memorySleep, memoryCurrentTime, memoryGetLastError
};

int sqlite3_os_init(void) {
    return sqlite3_vfs_register(&memoryVFS, 1);
}

int sqlite3_os_end(void) {
    return SQLITE_OK;
}

#include "demo.c"
