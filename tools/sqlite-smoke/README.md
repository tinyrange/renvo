# SQLite C backend smoke test

This local smoke test builds the official SQLite 3.53.4 amalgamation with
Renvo's C frontend and Linux/x86-64 backend, opens an in-memory database, runs
two queries through `sqlite3_exec`, and checks their callback output.

Run it from the repository root:

```sh
tools/sqlite-smoke/run
```

The first run downloads and verifies the pinned amalgamation, then takes a few
minutes to compile it. Downloads and build products stay under
`sandbox/sqlite-smoke`. Set `RENVO_SQLITE_SMOKE_WORK` to use another cache and
output directory. The script requires `curl`, `openssl`, and `unzip`.

This is deliberately a manual smoke test rather than a `tools/check` or CI
gate. It establishes that a substantial real C codebase compiles and executes,
but it is not yet a claim of complete SQLite support. In particular, this
initial test uses a small in-memory VFS and literal `SELECT` statements; broader
DDL, compound-query, filesystem, concurrency, and extension coverage remains
future work.
