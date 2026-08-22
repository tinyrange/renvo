#!/usr/bin/env python3

import os
import selectors
import subprocess
import sys
import time


def main() -> int:
    if len(sys.argv) != 4:
        print("usage: vmtest.py QEMU xv6.img fs.img", file=sys.stderr)
        return 2

    qemu, kernel_image, fs_image = sys.argv[1:]
    command = [
        qemu,
        "-nographic",
        "-monitor",
        "none",
        "-serial",
        "stdio",
        "-smp",
        "1",
        "-m",
        "512",
        "-drive",
        f"file={kernel_image},index=0,media=disk,format=raw",
        "-drive",
        f"file={fs_image},index=1,media=disk,format=raw",
    ]
    process = subprocess.Popen(
        command,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        bufsize=0,
    )
    assert process.stdin is not None
    assert process.stdout is not None
    selector = selectors.DefaultSelector()
    selector.register(process.stdout, selectors.EVENT_READ)
    deadline = time.monotonic() + int(os.environ.get("RENVO_XV6_TIMEOUT", "180"))
    transcript = bytearray()
    sent = False
    passed = False
    try:
        while time.monotonic() < deadline:
            if process.poll() is not None:
                break
            events = selector.select(timeout=1)
            for key, _ in events:
                chunk = os.read(key.fileobj.fileno(), 4096)
                if not chunk:
                    continue
                sys.stdout.buffer.write(chunk)
                sys.stdout.buffer.flush()
                transcript.extend(chunk)
                if not sent and b"init: starting sh\n$ " in transcript:
                    process.stdin.write(b"usertests\n")
                    process.stdin.flush()
                    sent = True
                if b"ALL TESTS PASSED" in transcript:
                    passed = True
                    return 0
        if not sent:
            print("xv6 VM did not reach its shell", file=sys.stderr)
        elif not passed:
            print("xv6 usertests did not report ALL TESTS PASSED", file=sys.stderr)
        return 1
    finally:
        if process.poll() is None:
            process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()
        if passed:
            print("\nxv6 vm test: PASS")


if __name__ == "__main__":
    raise SystemExit(main())
