#!/usr/bin/env python3
"""Read-only, allowlisted RFE demo server. No downloads or subprocesses."""
import hashlib
import http.client
import http.server
import json
import os
from pathlib import Path
import stat
import sys
import threading
from urllib.parse import urlsplit

ROOT = Path.cwd()
BIND = "100.66.0.124"
PORT = 8081
ARCHES = {"aarch64": "linux-arm64-machine", "x86_64": "linux-amd64-machine"}
CSP = "default-src 'none'; script-src 'unsafe-inline' 'wasm-unsafe-eval'; worker-src blob:; connect-src 'self'; style-src 'unsafe-inline'; frame-src 'self'; frame-ancestors 'self'; base-uri 'none'; form-action 'none'"


def regular(relative):
    path = ROOT
    for part in Path(relative).parts:
        path = path / part
        if path.is_symlink():
            raise ValueError("Symlink not allowed: " + relative)
    if not path.is_file():
        raise ValueError("Missing regular demo asset: " + relative)
    return path


def sha(path):
    with path.open("rb") as source:
        return hashlib.file_digest(source, "sha256").hexdigest()


def load_json(relative):
    return json.loads(regular(relative).read_text())


def prepare():
    pins = load_json("tools/rfe-linux/inputs.json")
    routes = {}
    public = {"service": "renvo-rfe-demo", "alpine": pins["version"], "architectures": {}}
    def add(url, relative, kind, expected=None):
        path = regular(relative)
        digest = sha(path)
        if expected is not None and digest != expected:
            raise ValueError("SHA-256 mismatch: " + relative)
        metadata = path.stat()
        routes[url] = (path, kind, digest, (metadata.st_dev, metadata.st_ino, metadata.st_size, metadata.st_mtime_ns))
        return metadata.st_size
    add("/", "tools/rfe-browser/index.html", "text/html; charset=utf-8")
    for arch, machine in ARCHES.items():
        prefix = "sandbox/rfe-linux/" + arch + "/"
        manifest = load_json(prefix + "manifest.json")
        if manifest["architecture"] != arch or manifest["alpine"] != pins["version"] or manifest["inputs"] != pins["architectures"][arch]:
            raise ValueError("Alpine minirootfs fixture provenance mismatch: " + arch)
        build = load_json("sandbox/rfe-browser/" + arch + ".html.json")
        if build["format"] != 1 or build["machine"] != machine or build["target"] != "browser/wasm32" or build["arena"] != 536870912:
            raise ValueError("Invalid browser build stamp: " + arch)
        sizes = {}
        sizes["html"] = add("/" + arch + "/", "sandbox/rfe-browser/" + arch + ".html", "text/html; charset=utf-8", build["html_sha256"])
        sizes["kernel"] = add("/" + arch + "/kernel", prefix + "kernel", "application/octet-stream", manifest["kernel_sha256"])
        sizes["initramfs"] = add("/" + arch + "/initramfs.cpio.gz", prefix + "initramfs.cpio.gz", "application/gzip", manifest["initramfs_sha256"])
        public["architectures"][arch] = {"sizes": sizes, "kernel_sha256": manifest["kernel_sha256"], "initramfs_sha256": manifest["initramfs_sha256"]}
    builds = [load_json("sandbox/rfe-browser/" + arch + ".html.json") for arch in ARCHES]
    if len({b["sources_sha256"] for b in builds}) != 1 or len({b["compiler_sha256"] for b in builds}) != 1:
        raise ValueError("Browser artifacts do not share reviewed sources/compiler")
    return routes, public


class Server(http.server.ThreadingHTTPServer):
    daemon_threads = True
    allow_reuse_address = False
    request_queue_size = 8
    def __init__(self, address, routes, public):
        self.routes, self.public = routes, public
        self.slots = threading.BoundedSemaphore(8)
        super().__init__(address, Handler)
    def get_request(self):
        request, address = super().get_request()
        request.settimeout(30)
        return request, address
    def process_request(self, request, address):
        if not self.slots.acquire(blocking=False):
            request.close()
            return
        try:
            super().process_request(request, address)
        except BaseException:
            self.slots.release()
            raise
    def process_request_thread(self, request, address):
        try:
            super().process_request_thread(request, address)
        finally:
            self.slots.release()


class Handler(http.server.BaseHTTPRequestHandler):
    server_version = "RenvoDemo/1"
    sys_version = ""
    def log_message(self, *args):
        pass  # Do not accumulate per-request logs in the background process.
    def send_headers(self, code, length, kind, etag=None):
        self.send_response(code)
        self.send_header("Content-Type", kind)
        self.send_header("Content-Length", str(length))
        self.send_header("Cache-Control", "no-cache")
        self.send_header("Content-Security-Policy", CSP)
        self.send_header("X-Content-Type-Options", "nosniff")
        self.send_header("Referrer-Policy", "no-referrer")
        self.send_header("Cross-Origin-Resource-Policy", "same-origin")
        self.send_header("Connection", "close")
        if etag:
            self.send_header("ETag", '"' + etag + '"')
        self.end_headers()
    def do_HEAD(self):
        self.serve(False)
    def do_GET(self):
        self.serve(True)
    def serve(self, body):
        try:
            parsed = urlsplit(self.path)
            if parsed.scheme or parsed.netloc:
                self.send_headers(400, 0, "text/plain")
                return
            path = parsed.path
            if path == "/healthz":
                data = (json.dumps(self.server.public, sort_keys=True) + "\n").encode()
                self.send_headers(200, len(data), "application/json")
                if body:
                    self.wfile.write(data)
                return
            route = self.server.routes.get(path)
            if route is None:
                self.send_headers(404, 0, "text/plain")
                return
            asset, kind, digest, signature = route
            fd = os.open(asset, os.O_RDONLY | os.O_NOFOLLOW)
            with os.fdopen(fd, "rb") as source:
                s = os.fstat(source.fileno())
                if not stat.S_ISREG(s.st_mode) or (s.st_dev, s.st_ino, s.st_size, s.st_mtime_ns) != signature:
                    self.send_headers(503, 0, "text/plain")
                    return
                self.send_headers(200, s.st_size, kind, digest)
                if body:
                    while True:
                        block = source.read(256 * 1024)
                        if not block:
                            break
                        self.wfile.write(block)
        except (BrokenPipeError, ConnectionResetError, TimeoutError):
            return


def request(host, port, method, path):
    connection = http.client.HTTPConnection(host, port, timeout=10)
    try:
        connection.request(method, path)
        response = connection.getresponse()
        data = response.read()
        return response.status, dict(response.getheaders()), data
    finally:
        connection.close()


def check(routes, public):
    server = Server(("127.0.0.1", 0), routes, public)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        port = server.server_address[1]
        for url, (_, kind, digest, signature) in routes.items():
            status, headers, data = request("127.0.0.1", port, "GET", url)
            assert status == 200 and len(data) == signature[2] and hashlib.sha256(data).hexdigest() == digest, url
            assert headers["Content-Type"] == kind and headers["X-Content-Type-Options"] == "nosniff", url
            status, headers, data = request("127.0.0.1", port, "HEAD", url)
            assert status == 200 and not data and int(headers["Content-Length"]) == signature[2], url
        for path in ["/AGENTS.star", "/.git/config", "/../AGENTS.star", "/aarch64/../../AGENTS.star", "/%2e%2e/AGENTS.star", "/aarch64/kernel/", "/x86_64/manifest.json"]:
            assert request("127.0.0.1", port, "GET", path)[0] == 404, path
        assert request("127.0.0.1", port, "POST", "/")[0] == 501
        status, headers, data = request("127.0.0.1", port, "GET", "/healthz")
        assert status == 200 and json.loads(data) == public
        print("PASS: both Alpine minirootfs fixtures, artifact hashes, GET/HEAD, allowlist/traversal/write rejection", flush=True)
    finally:
        server.shutdown()
        server.server_close()
        thread.join()


def main():
    if sys.argv[1:] not in [["--check"], ["--serve"], ["--probe"]]:
        raise ValueError("Expected --check, --serve, or --probe")
    if sys.argv[1:] == ["--probe"]:
        status, _, data = request(BIND, PORT, "GET", "/healthz")
        if status != 200 or json.loads(data).get("service") != "renvo-rfe-demo":
            raise ValueError("Unexpected service response")
        for path in ["/", "/aarch64/", "/aarch64/kernel", "/aarch64/initramfs.cpio.gz", "/x86_64/", "/x86_64/kernel", "/x86_64/initramfs.cpio.gz"]:
            status, headers, data = request(BIND, PORT, "HEAD", path)
            if status != 200 or data or int(headers.get("Content-Length", "0")) <= 0:
                raise ValueError("Serving check failed: " + path)
        print("PASS: http://" + BIND + ":" + str(PORT) + "/ serves both architectures and preconfigured boot assets")
        return
    routes, public = prepare()
    if sys.argv[1:] == ["--check"]:
        check(routes, public)
        return
    with Server((BIND, PORT), routes, public) as server:
        print("READY http://" + BIND + ":" + str(PORT) + "/ Alpine " + public["alpine"] + " minirootfs (aarch64, x86_64)", flush=True)
        server.serve_forever()


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        pass
    except Exception as error:
        print("RFE demo server: " + str(error), file=sys.stderr, flush=True)
        sys.exit(1)
