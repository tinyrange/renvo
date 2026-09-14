"""Compile Gemma's tokenizer.json into direct native lookup tables (stdlib only)."""
import argparse
from array import array
import json
from pathlib import Path
import struct
import sys


def words(values):
    data = array("I", values)
    if sys.byteorder != "little":
        data.byteswap()
    return data.tobytes()


def prepare(model, output):
    tokenizer = json.loads((model / "tokenizer.json").read_text())
    bpe = tokenizer["model"]
    required = {"type": "BPE", "dropout": None, "unk_token": "<unk>",
                "continuing_subword_prefix": None, "end_of_word_suffix": None,
                "fuse_unk": True, "byte_fallback": True, "ignore_merges": False}
    if any(bpe.get(k) != v for k, v in required.items()):
        raise ValueError("Unsupported BPE configuration")
    if tokenizer["normalizer"] != {"type": "Replace", "pattern": {"String": " "}, "content": "▁"}:
        raise ValueError("Unsupported normalizer")
    if tokenizer["pre_tokenizer"] != {"type": "Split", "pattern": {"String": " "}, "behavior": "MergedWithPrevious", "invert": False}:
        raise ValueError("Unsupported pre-tokenizer")
    if tokenizer["decoder"] != {"type": "Sequence", "decoders": [
        {"type": "Replace", "pattern": {"String": "▁"}, "content": " "},
        {"type": "ByteFallback"}, {"type": "Fuse"}]}:
        raise ValueError("Unsupported decoder")
    vocab = bpe["vocab"]
    if sorted(vocab.values()) != list(range(262144)):
        raise ValueError("Expected dense Gemma 270M vocabulary")
    for added in tokenizer["added_tokens"]:
        if added["content"] not in vocab:
            if added["id"] != len(vocab):
                raise ValueError("Non-contiguous added vocabulary")
            vocab[added["content"]] = added["id"]
    for token, expected in [("<bos>", 2), ("<eos>", 1), ("<start_of_turn>", 105), ("<end_of_turn>", 106)]:
        if vocab.get(token) != expected:
            raise ValueError("Unsupported chat special tokens")
    runes = array("I", [0]) * 0x110000
    flags = bytearray(len(vocab))
    pieces = [b""] * len(vocab)
    byte_ids = [vocab[f"<0x{i:02X}>"] for i in range(256)]
    for token, token_id in vocab.items():
        if len(token) == 1:
            runes[ord(token)] = token_id + 1
        pieces[token_id] = token.replace("▁", " ").encode()
    for byte, token_id in enumerate(byte_ids):
        flags[token_id] = 2
        pieces[token_id] = bytes([byte])
    # child, sibling, byte, terminal ID + 1; node zero is the root.
    nodes = [[0, 0, 0, 0]]
    edges = {}
    for added in tokenizer["added_tokens"]:
        if any(added[k] for k in ("single_word", "lstrip", "rstrip", "normalized")):
            raise ValueError("Unsupported added-token matching flags")
        token_id = added["id"]
        if vocab.get(added["content"]) != token_id or not added["content"]:
            raise ValueError("Invalid added token")
        if added["special"]:
            flags[token_id] |= 1
        node = 0
        for byte in added["content"].encode():
            key = (node, byte)
            if key not in edges:
                edges[key] = len(nodes)
                nodes.append([0, nodes[node][0], byte, 0])
                nodes[node][0] = edges[key]
            node = edges[key]
        nodes[node][3] = token_id + 1
    slots = 1
    while slots < 2 * len(bpe["merges"]):
        slots *= 2
    pairs = array("I", [0]) * (slots * 4)
    for rank, (left, right) in enumerate(bpe["merges"]):
        a, b, merged = vocab[left], vocab[right], vocab[left + right]
        slot = ((a * 0x9E3779B1) ^ (b * 0x85EBCA77)) & (slots - 1)
        while pairs[slot * 4 + 2]:
            if pairs[slot * 4] == a and pairs[slot * 4 + 1] == b:
                raise ValueError("Duplicate merge pair")
            slot = (slot + 1) & (slots - 1)
        pairs[slot * 4:slot * 4 + 4] = array("I", [a, b, rank + 1, merged])
    offsets = [0]
    for piece in pieces:
        offsets.append(offsets[-1] + len(piece))
    header = [0x31544D47, 1, 0, len(vocab), slots, len(nodes)] + [0] * 10
    sections = [words(runes), words(pairs), words(v for n in nodes for v in n),
                words(offsets), bytes(flags), b"".join(pieces), words(byte_ids)]
    with output.open("wb") as stream:
        stream.write(bytes(64))
        for index, section in enumerate(sections):
            header[6 + index] = stream.tell()
            stream.write(section)
        header[2] = stream.tell()
        stream.seek(0)
        stream.write(struct.pack("<16I", *header))
    print(f"Packed {output}: {header[2]:,} bytes; {len(bpe['merges']):,} merges")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("model", type=Path)
    parser.add_argument("output", type=Path)
    args = parser.parse_args()
    prepare(args.model, args.output)
