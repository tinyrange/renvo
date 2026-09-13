#!/usr/bin/env python3
"""Verify real TCP echo traffic on macOS, binding to the wired interface."""
import argparse
import os
import socket
import subprocess
import time

parser = argparse.ArgumentParser()
parser.add_argument('--interface', default='en5')
parser.add_argument('--address', default='169.254.180.4')
parser.add_argument('--port', type=int, default=4242)
parser.add_argument('--connections', type=int, default=3)
args = parser.parse_args()

# A PHY reset briefly removes macOS's automatic link-local address. Wait for
# it to return, and explicitly scope the socket away from competing Wi-Fi routes.
deadline = time.monotonic() + 60
while True:
    result = subprocess.run(['ipconfig', 'getifaddr', args.interface], capture_output=True, text=True)
    local = result.stdout.strip()
    if result.returncode == 0 and local:
        break
    if time.monotonic() >= deadline:
        raise SystemExit(f'No IPv4 address on {args.interface}; configure a compatible address first')
    time.sleep(.5)

print(f'{args.interface} {local} -> {args.address}:{args.port}', flush=True)
for connection in range(args.connections):
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        # IP_BOUND_IF is 25 in the macOS SDK's netinet/in.h.
        sock.setsockopt(socket.IPPROTO_IP, 25, socket.if_nametoindex(args.interface))
        sock.bind((local, 0))
        sock.settimeout(15)
        sock.connect((args.address, args.port))
        for size in (1, 31, 512, 1024, 8192, 65536):
            sent = os.urandom(size)
            started = time.monotonic()
            sock.sendall(sent)
            received = bytearray()
            while len(received) < size:
                part = sock.recv(size-len(received))
                if not part:
                    raise RuntimeError('Unexpected EOF')
                received.extend(part)
            if received != sent:
                raise RuntimeError(f'Payload mismatch on connection {connection+1}')
            print(f'PASS connection {connection+1}: {size} bytes in {time.monotonic()-started:.3f}s', flush=True)
        sock.shutdown(socket.SHUT_WR)
        if sock.recv(1) != b'':
            raise RuntimeError('Expected orderly EOF')
        print('PASS orderly TCP close', flush=True)
