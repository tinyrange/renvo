"""Exercise the board's HTTP server over a real TCP connection."""
import concurrent.futures
import http.client
import json
import socket
import sys
import time

address = sys.argv[1]


def request(path, method="GET", expected=200):
    conn = http.client.HTTPConnection(address, 80, timeout=10)
    conn.request(method, path)
    response = conn.getresponse()
    body = response.read()
    assert response.status == expected, (path, response.status)
    assert response.getheader("Connection") == "close"
    if method != "HEAD" and expected != 204:
        assert len(body) == int(response.getheader("Content-Length"))
    conn.close()
    return body


assert b"Unit PoE-P4" in request("/")
status = json.loads(request("/status.json"))
assert status["ip"] == address, status
assert request("/healthz") == b"ok\n"
assert request("/", "HEAD") == b""
request("/missing", expected=404)
request("/", "POST", 405)
request("/favicon.ico", expected=204)
with socket.create_connection((address, 80), timeout=10) as sock:
    sock.sendall(b"GET /healthz HTTP/1.1\r\nHo")
    time.sleep(.1)
    sock.sendall(b"st: board\r\n\r\n")
    response = http.client.HTTPResponse(sock)
    response.begin()
    assert response.status == 200 and response.read() == b"ok\n"
with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
    bodies = list(pool.map(lambda _: request("/"), range(40)))
    assert all(b"</html>" in body for body in bodies)
print("PASS: routes, HEAD, error responses, split request and 40 concurrent-page requests")
print(json.dumps(json.loads(request("/status.json")), indent=2))
