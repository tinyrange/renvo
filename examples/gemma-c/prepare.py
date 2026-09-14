"""Pack the local Gemma 3 270M BF16 tensors for the small C runtime."""
import argparse
import json
import math
import struct
from pathlib import Path

FIELDS = ["input_layernorm.weight", "self_attn.q_proj.weight",
          "self_attn.k_proj.weight", "self_attn.v_proj.weight",
          "self_attn.q_norm.weight", "self_attn.k_norm.weight",
          "self_attn.o_proj.weight", "post_attention_layernorm.weight",
          "pre_feedforward_layernorm.weight", "mlp.gate_proj.weight",
          "mlp.up_proj.weight", "mlp.down_proj.weight",
          "post_feedforward_layernorm.weight"]

def prepare(model, output):
    cfg = json.loads((model / "config.json").read_text())
    dims = [cfg[k] for k in ("hidden_size", "intermediate_size", "num_hidden_layers",
                            "num_attention_heads", "num_key_value_heads", "head_dim", "vocab_size")]
    if dims != [640, 2048, 18, 4, 1, 256, 262144]:
        raise ValueError("This first runtime supports Gemma 3 270M dimensions only")
    if cfg.get("rope_scaling") or cfg.get("attention_bias") or cfg.get("final_logit_softcapping") or cfg.get("attn_logit_softcapping"):
        raise ValueError("Unsupported attention/rotary configuration")
    if cfg["hidden_activation"] != "gelu_pytorch_tanh":
        raise ValueError("Unsupported activation")
    header = [0] * 512
    header[:12] = [0x31434d47, 1, 0, *dims, cfg["sliding_window"], 1024]
    header[12] = struct.unpack("<I", struct.pack("<f", cfg["rms_norm_eps"]))[0]
    header[13] = struct.unpack("<I", struct.pack("<f", cfg["query_pre_attn_scalar"] ** -0.5))[0]
    with (model / "model.safetensors").open("rb") as src, output.open("wb") as dst:
        n = struct.unpack("<Q", src.read(8))[0]
        tensors = json.loads(src.read(n))
        data_start = 8 + n
        dst.write(bytes(2048))
        def tensor(name, slot, shape):
            entry = tensors[name]
            if entry["dtype"] != "BF16" or entry["shape"] != shape:
                raise ValueError(f"Unexpected tensor {name}: {entry}")
            begin, end = entry["data_offsets"]
            if end - begin != math.prod(shape) * 2:
                raise ValueError(f"Invalid extent for {name}")
            header[slot] = dst.tell()
            src.seek(data_start + begin)
            remaining = end - begin
            while remaining:
                block = src.read(min(remaining, 1024 * 1024))
                if not block:
                    raise ValueError("Truncated safetensors")
                dst.write(block)
                remaining -= len(block)
        tensor("model.embed_tokens.weight", 32, [262144, 640])
        tensor("model.norm.weight", 33, [640])
        shapes = [[640], [1024, 640], [256, 640], [256, 640], [256], [256],
                  [640, 1024], [640], [640], [2048, 640], [2048, 640], [640, 2048], [640]]
        for layer in range(18):
            for i, (name, shape) in enumerate(zip(FIELDS, shapes)):
                tensor(f"model.layers.{layer}.{name}", 64 + layer * 16 + i, shape)
            kind = cfg["layer_types"][layer]
            if kind not in ("sliding_attention", "full_attention"):
                raise ValueError(f"Unsupported attention type {kind}")
            header[64 + layer * 16 + 13] = int(kind == "sliding_attention")
        for slot, base in [(34, cfg["rope_local_base_freq"]), (35, cfg["rope_theta"])]:
            header[slot] = dst.tell()
            dst.write(struct.pack("<128f", *(base ** (-2 * i / 256) for i in range(128))))
        header[2] = dst.tell()
        dst.seek(0)
        dst.write(struct.pack("<512I", *header))
    print(f"Packed {output}: {header[2]:,} bytes")

if __name__ == "__main__":
    p = argparse.ArgumentParser()
    p.add_argument("model", type=Path)
    p.add_argument("output", type=Path)
    args = p.parse_args()
    prepare(args.model, args.output)
