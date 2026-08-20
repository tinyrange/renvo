# xv6 i386 VM gate

`./tools/check xv6` (or `./tools/xv6/run` directly) checks out the pinned
`mit-pdos/xv6-public` revision, builds
all kernel and user C translation units with `renvo cc -m32`, links them with
the system i386 linker, boots the images in QEMU, and runs `usertests`. Success
requires xv6's canonical `ALL TESTS PASSED` output.

The host C compiler is used only for xv6's size-constrained boot block and the
host `mkfs` utility. The system assembler handles the existing `.S` files.

The checked patch makes three source-level portability corrections: kernel and
user `printf` use standard builtin varargs instead of deriving unnamed arguments
from a named parameter's address, and `getcallerpcs` asks for the caller frame
instead of assuming GCC's exact parameter-frame layout. These preserve xv6's
semantics without encoding compiler-specific stack layouts in Renvo.

Set `RENVO_XV6_SOURCE` to an existing xv6 Git clone to avoid a network fetch,
`RENVO_XV6_FRONTEND` to test an existing executable, or `RENVO_XV6_TIMEOUT` to
change the 180-second VM timeout.
