#!/usr/bin/env python3
"""Optional native launcher and CPU reference verification harness."""
import argparse
from pathlib import Path
import struct
import subprocess
import tempfile
import time

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--model", type=Path, required=True)
    parser.add_argument("--weights", type=Path, required=True)
    parser.add_argument("--runtime", type=Path, required=True)
    parser.add_argument("--tokenizer", type=Path, help="Packed tokenizer; defaults to tokenizer.bin beside weights")
    parser.add_argument("--prompt", default="What is the capital of France?")
    parser.add_argument("--steps", type=int, default=32)
    parser.add_argument("--verify", action="store_true", help="Compare prompt logits with CPU PyTorch")
    args = parser.parse_args()
    tables = args.tokenizer or args.weights.with_name("tokenizer.bin")
    if not args.verify:
        subprocess.run([str(args.runtime.resolve()), "--chat", str(args.weights.resolve()),
                        str(tables.resolve()), args.prompt, str(args.steps)], check=True)
        return
    from transformers import AutoTokenizer

    tokenizer = AutoTokenizer.from_pretrained(args.model, local_files_only=True)
    encoded = tokenizer.apply_chat_template(
        [{"role": "user", "content": args.prompt}], tokenize=True, add_generation_prompt=True
    )
    expected_ids = encoded if isinstance(encoded, list) else encoded["input_ids"]
    native_ids = subprocess.check_output([str(args.runtime.resolve()), "--tokenize", str(tables.resolve()), args.prompt], text=True)
    ids = [int(token) for token in native_ids.split()]
    assert ids == expected_ids, "native prompt tokenizer differs from reference"
    if not 0 <= args.steps <= 1024 - len(ids):
        parser.error("prompt plus steps must fit the 1024-token context")
    with tempfile.TemporaryDirectory(prefix="renvo-gemma-") as temporary:
        prompt = Path(temporary) / "prompt.ids"
        logits = Path(temporary) / "logits.bin"
        prompt.write_bytes(struct.pack(f"<{len(ids)}I", *ids))
        command = [str(args.runtime.resolve()), str(args.weights.resolve()), str(prompt), str(args.steps)]
        if args.verify:
            command.append(str(logits))
        start = time.monotonic()
        result = subprocess.run(command, check=True, text=True, capture_output=True)
        elapsed = time.monotonic() - start
        generated = [int(token) for token in result.stdout.split()]
        output_ids = Path(temporary) / "output.ids"
        output_ids.write_bytes(struct.pack(f"<{len(generated)}I", *generated))
        decoded = subprocess.check_output([str(args.runtime.resolve()), "--decode", str(tables.resolve()), str(output_ids)]).decode()
        assert decoded == tokenizer.decode(generated, skip_special_tokens=True), "native decoding differs"
        print(decoded)
        print(f"{len(ids)} prompt tokens; {len(generated)} generated tokens; {elapsed:.2f}s total")
        if args.verify:
            import numpy as np
            import torch
            from transformers import AutoModelForCausalLM

            torch.set_num_threads(4)
            reference = AutoModelForCausalLM.from_pretrained(
                args.model, local_files_only=True, dtype=torch.float32, attn_implementation="eager"
            ).eval()
            with torch.inference_mode():
                expected = reference(torch.tensor([ids])).logits[0, -1].numpy()
            actual = np.fromfile(logits, dtype="<f4")
            np.testing.assert_allclose(actual, expected, atol=0.015, rtol=0.002)
            assert actual.argmax() == expected.argmax(), "greedy token differs"
            print(f"Reference passed: max logit error {np.max(np.abs(actual-expected)):.6g}; "
                  f"next token {actual.argmax()}")
            if generated:
                with torch.inference_mode():
                    sequence = reference.generate(
                        torch.tensor([ids]), attention_mask=torch.ones(1, len(ids), dtype=torch.long),
                        do_sample=False, max_new_tokens=args.steps, eos_token_id=[1, 106], pad_token_id=0,
                    )[0, len(ids):].tolist()
                assert generated == sequence, f"greedy sequence differs: {generated} != {sequence}"
                print(f"All {len(generated)} generated tokens match the reference")


if __name__ == "__main__":
    main()
