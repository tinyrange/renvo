"""Differential tests of the Renvo executable against the checkpoint tokenizer."""
import argparse
import json
from pathlib import Path
import random
import struct
import subprocess
import tempfile
import time

from tokenizers import Tokenizer
from transformers import AutoTokenizer


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--model", type=Path, required=True)
    parser.add_argument("--tokenizer", type=Path, required=True)
    parser.add_argument("--runtime", type=Path, required=True)
    args = parser.parse_args()
    reference = Tokenizer.from_file(str(args.model / "tokenizer.json"))
    chat = AutoTokenizer.from_pretrained(args.model, local_files_only=True)
    data = json.loads((args.model / "tokenizer.json").read_text())
    command = [str(args.runtime.resolve())]
    tables = str(args.tokenizer.resolve())
    cases = ["", " ", "   ", "hello", "Hello world!", "What is the capital of France?",
             "a" * 512, "abab" * 128, " " * 1024, "\n" * 90, "\t" * 90,
             "  tabs\tand\r\nnewlines  ", "▁literal▁marker", "<bos><eos><end_of_turn>",
             "<image_soft_token>", "<unused12><unused1>", "<start_of_turn>user\nhi",
             "café cafe\u0301", "你好，世界！", "こんにちは世界", "مرحبا بالعالم", "नमस्ते दुनिया",
             "👩🏽‍💻 🦄 🫠", "\U0010ffff\U00040000\u0378", "\u00a0\u2003trim me\u3000",
             "\x1c\x1d\x1e\x1ftrim\x85", "def f(x):\n    return x * 1.5\n"]
    rng = random.Random(7301)
    alphabet = "abcxyz ABCXYZ019!?\n\t▁é中👩🦄\u0378\u00a0"
    cases += ["".join(rng.choices(alphabet, k=rng.randrange(1, 240))) for _ in range(40)]
    # Exercise every added-token terminal, including shared prefixes and aliases.
    added = [entry["content"] for entry in data["added_tokens"]]
    cases += ["x" + "y".join(added[i:i+64]) + "z" for i in range(0, len(added), 64)]
    started = time.monotonic()
    for text in cases:
        actual = subprocess.check_output(command + ["--encode", tables, text], text=True)
        ids = [int(value) for value in actual.split()]
        expected = reference.encode(text, add_special_tokens=False).ids
        assert ids == expected, f"encoding mismatch for {text!r}: {ids} != {expected}"
    for text in cases[:27]:
        actual = subprocess.check_output(command + ["--tokenize", tables, text], text=True)
        encoded = chat.apply_chat_template([{"role": "user", "content": text}], tokenize=True, add_generation_prompt=True)
        expected = encoded if isinstance(encoded, list) else encoded["input_ids"]
        assert [int(value) for value in actual.split()] == expected, f"chat mismatch for {text!r}"
    vocab = reference.get_vocab()
    byte_id = lambda value: vocab[f"<0x{value:02X}>"]
    sequences = [list(range(reference.get_vocab_size())),
                 [byte_id(b) for b in "👩🏽‍💻".encode()],
                 [byte_id(b) for b in [65, 255, 66]],
                 [byte_id(0xc3), 2, byte_id(0xa9)],
                 [byte_id(0xc3), vocab["hello"], byte_id(0xa9)],
                 [byte_id(0xf0), byte_id(0x9f)], [1, 2, 106, 262144]]
    sequences += [[rng.randrange(reference.get_vocab_size()) for _ in range(100)] for _ in range(20)]
    with tempfile.TemporaryDirectory(prefix="renvo-tokenizer-test-") as directory:
        path = Path(directory) / "ids.bin"
        for ids in sequences:
            path.write_bytes(struct.pack(f"<{len(ids)}I", *ids))
            actual = subprocess.check_output(command + ["--decode", tables, str(path)]).decode()
            expected = reference.decode(ids, skip_special_tokens=True)
            assert actual == expected, f"decoding mismatch for IDs {ids[:20]}: {actual[:100]!r} != {expected[:100]!r}"
        failures = [
            (["--encode", tables, "a" * 65537], b"exceeds"),
            (["--encode", tables, b"\xff"], b"valid UTF-8"),
            (["--chat", "missing-model.bin", tables, "hello", "4294967296"], b"generation budget"),
            (["--chat", "missing-model.bin", tables, "<image_soft_token>", "1"], b"outside the text model"),
        ]
        path.write_bytes(struct.pack("<I", 0xffffffff))
        failures.append((["--decode", tables, str(path)], b"Invalid output token"))
        for arguments, message in failures:
            result = subprocess.run(command + arguments, capture_output=True)
            assert result.returncode == 2 and message in result.stderr, (arguments[:2], result)
    print(f"PASS: {len(cases)} encodings, 27 chat templates, {len(sequences)} decodings "
          f"(including all vocabulary IDs), {time.monotonic()-started:.2f}s")


if __name__ == "__main__":
    main()
