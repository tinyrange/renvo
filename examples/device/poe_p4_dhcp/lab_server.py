#!/usr/bin/env python3
"""Short-lived, one-MAC DHCP fixture for an isolated direct macOS cable.

Uses UDP, not raw packets. Never answers other clients. Deliberately offers
an address on the Mac's existing link-local subnet for this isolated test.
This is a test fixture, not a DHCP service for a regular LAN.
"""
import argparse
import socket
import struct
import time

parser = argparse.ArgumentParser()
parser.add_argument('--interface', default='en5')
parser.add_argument('--mac', default='80:f1:b2:d1:4c:be')
parser.add_argument('--server', default='169.254.180.218')
parser.add_argument('--address', default='169.254.180.44')
parser.add_argument('--duration', type=int, default=90)
parser.add_argument('--skip-renewals', action='store_true')
parser.add_argument('--expire', action='store_true')
args = parser.parse_args()
mac = bytes.fromhex(args.mac.replace(':', ''))
server = socket.inet_aton(args.server)
address = socket.inet_aton(args.address)
cookie = bytes.fromhex('63825363')

def options(data):
    result = {}
    offset = 240
    while offset < len(data):
        code = data[offset]
        offset += 1
        if code == 255:
            return result
        if code == 0:
            continue
        if offset >= len(data):
            return {}
        size = data[offset]
        offset += 1
        if offset + size > len(data):
            return {}
        result[code] = data[offset:offset+size]
        offset += size
    return {}

def option(code, data):
    return bytes((code, len(data))) + data

with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as sock:
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_BROADCAST, 1)
    sock.setsockopt(socket.IPPROTO_IP, 25, socket.if_nametoindex(args.interface))
    sock.bind(('', 67))
    sock.settimeout(.5)
    print(f'DHCP fixture {args.interface}: only {args.mac}, offer {args.address}, lease 30s/T1 12s/T2 24s', flush=True)
    until = time.monotonic() + args.duration
    initial_ack = False
    renew_ack_after = 0
    while time.monotonic() < until:
        try:
            data, peer = sock.recvfrom(2048)
        except socket.timeout:
            continue
        if len(data) < 240 or data[:3] != b'\x01\x01\x06' or data[28:34] != mac or data[236:240] != cookie:
            continue
        opts = options(data)
        kind = opts.get(53)
        renewing = data[12:16] != b'\0'*4
        if kind == b'\x04':
            print('DECLINE: address conflict reported', flush=True)
            continue
        if kind not in (b'\x01', b'\x03'):
            continue
        if kind == b'\x03' and not renewing and (opts.get(50) != address or opts.get(54) != server):
            continue
        if renewing and data[12:16] != address:
            continue
        if renewing and (args.expire or args.skip_renewals and time.monotonic() < renew_ack_after):
            print('Withholding renewal ACK for lifecycle test', flush=True)
            continue
        if args.expire and initial_ack:
            print('Withholding all replies after first lease for expiry test', flush=True)
            continue
        reply_kind = 2 if kind == b'\x01' else 5
        reply = bytearray(240)
        reply[:3] = b'\x02\x01\x06'
        reply[4:12] = data[4:12]
        reply[16:20] = address
        reply[20:24] = server
        reply[28:34] = mac
        reply[236:240] = cookie
        reply += option(53, bytes((reply_kind,))) + option(54, server)
        reply += option(1, socket.inet_aton('255.255.0.0'))
        reply += option(3, server) + option(6, server)
        for code, seconds in ((51, 30), (58, 12), (59, 24)):
            reply += option(code, struct.pack('!I', seconds))
        reply += b'\xff'
        reply += bytes(max(0, 300-len(reply)))
        destination = args.address if renewing else '255.255.255.255'
        try:
            sock.sendto(reply, (destination, 68))
        except OSError as error:
            print(f'Send deferred while interface acquires IPv4: {error}', flush=True)
            continue
        print(f'{"OFFER" if reply_kind == 2 else "ACK"} xid={data[4:8].hex()} ciaddr={socket.inet_ntoa(data[12:16])} to={destination}', flush=True)
        if reply_kind == 5:
            initial_ack = True
            renew_ack_after = time.monotonic() + 22
    print('DHCP fixture stopped', flush=True)
