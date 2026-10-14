import gzip
import importlib.machinery
import importlib.util
import io
from pathlib import Path
import stat
import struct
import sys
import tarfile
import tempfile
import unittest


sys.dont_write_bytecode = True
loader = importlib.machinery.SourceFileLoader("prepare", str(Path(__file__).with_name("prepare")))
spec = importlib.util.spec_from_loader(loader.name, loader)
prepare = importlib.util.module_from_spec(spec)
loader.exec_module(prepare)


def decode_newc(data):
    """Read records independently of the writer, checking bounds and padding."""
    records = {}
    offset = 0
    while True:
        assert data[offset:offset + 6] == b"070701"
        fields = [int(data[offset + 6 + 8 * i:offset + 14 + 8 * i], 16) for i in range(13)]
        offset += 110
        name = data[offset:offset + fields[11]]
        assert name[-1:] == b"\0"
        offset += len(name)
        aligned = (offset + 3) & ~3
        assert not any(data[offset:aligned])
        offset = aligned
        payload = data[offset:offset + fields[6]]
        assert len(payload) == fields[6]
        offset += len(payload)
        aligned = (offset + 3) & ~3
        assert not any(data[offset:aligned])
        offset = aligned
        if name == b"TRAILER!!!\0":
            assert not any(data[offset:])
            return records
        records[name[:-1].decode()] = (fields, payload)


class PrepareTests(unittest.TestCase):
    def test_newc_reproducible_links_and_devices(self):
        entries = {
            ".": prepare.node(stat.S_IFDIR | 0o755),
            "bin": prepare.node(stat.S_IFDIR | 0o755),
            "bin/sh": prepare.node(stat.S_IFLNK | 0o777, b"busybox"),
            "bin/busybox": prepare.node(stat.S_IFREG | 0o755, b"ELF fixture\0"),
            "bin/zalias": dict(prepare.node(stat.S_IFREG | 0o755), target="bin/busybox"),
            "dev": prepare.node(stat.S_IFDIR | 0o755),
            "dev/console": prepare.node(stat.S_IFCHR | 0o600, major=5, minor=1),
        }
        archive = prepare.cpio(entries)
        self.assertEqual(archive, prepare.cpio(dict(reversed(list(entries.items())))))
        self.assertEqual(len(archive) % 512, 0)
        records = decode_newc(archive)
        first, last = records["bin/busybox"], records["bin/zalias"]
        self.assertEqual(first[0][0], last[0][0])
        self.assertEqual(first[0][4], 2)
        self.assertEqual(first[1], b"")
        self.assertEqual(last[1], b"ELF fixture\0")
        self.assertEqual(records["bin/sh"][1], b"busybox")
        self.assertEqual(records["bin/sh"][0][1], stat.S_IFLNK | 0o777)
        self.assertEqual(records["dev/console"][0][9:11], [5, 1])
        self.assertTrue(all(record[0][5] == 0 for record in records.values()))

    def test_unsafe_paths(self):
        for name in ("/etc/passwd", "../escape", "a/../../escape", "a\0b"):
            with self.subTest(name=name), self.assertRaises(ValueError):
                prepare.clean_path(name)
        self.assertEqual(prepare.clean_path("./bin/sh"), "bin/sh")

    def test_tar_parent_and_hardlink_rejection(self):
        with tempfile.TemporaryDirectory() as temporary:
            path = Path(temporary) / "root.tar.gz"
            with tarfile.open(path, "w:gz") as archive:
                link = tarfile.TarInfo("bin")
                link.type, link.linkname = tarfile.SYMTYPE, "/tmp"
                archive.addfile(link)
                regular = tarfile.TarInfo("bin/program")
                regular.size = 1
                archive.addfile(regular, io.BytesIO(b"x"))
            with self.assertRaisesRegex(ValueError, "non-directory"):
                prepare.rootfs_entries(path)
        for links in ({"a": "b"}, {"a": "b", "b": "a"}):
            with self.assertRaises(ValueError):
                prepare.cpio({name: dict(prepare.node(stat.S_IFREG | 0o644), target=target)
                              for name, target in links.items()})

    def test_arm64_efi_zboot_payload(self):
        image = bytearray(128)
        image[56:60] = b"ARM\x64"
        payload = gzip.compress(image, mtime=0)
        header = bytearray(64)
        header[:8] = b"MZ\0\0zimg"
        struct.pack_into("<II", header, 8, 64, len(payload))
        header[24:29] = b"gzip\0"
        self.assertEqual(prepare.arm64_image(bytes(header) + payload + b"PE padding"), image)
        with self.assertRaises(ValueError):
            prepare.arm64_image(bytes(header) + payload[:-1])


if __name__ == "__main__":
    unittest.main()
