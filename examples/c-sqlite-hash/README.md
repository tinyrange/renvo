# SQLite-style C hash table

This project exercises `renvo make` and the Linux/amd64 object linker with a
small multi-file C library. The separate-chaining table and its single linked
iteration list follow the generic hash-table design used by early SQLite. The
demo uses a fixed arena so the project has no host libc dependency.

Build it from this directory:

```sh
renvo make
./app
```

The program exits successfully after inserting, replacing, finding, iterating,
and removing records. The Web IDE can run the same `renvo make` command from
its Terminal tab and download the resulting raw Linux executable.
