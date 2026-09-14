# Gemma 3 in C, compiled by Renvo

An experimental CPU runtime for the local **Gemma 3 270M instruction-tuned**
checkpoint. The transformer runs entirely in C compiled to a native executable
by Renvo. Go handles tokenization, the single-user chat template, decoding, and
bulk file I/O, compiled into the same executable. Normal inference needs no
Python or external libraries. Python is used for one-time preprocessing and
optional reference verification; PyTorch is only used for verification.

The runtime retains BF16 weights, uses float32 activations, and implements
18 transformer layers, grouped-query attention, per-head Q/K normalization,
local/global RoPE, sliding-window attention, GELU, a KV cache, and greedy decoding.
It follows the [Transformers Gemma 3 implementation](https://github.com/huggingface/transformers/tree/main/src/transformers/models/gemma3).

## Run on macOS Apple Silicon

From the repository root (requires Go and Python 3 for one-time setup):

```sh
mkdir -p sandbox/gemma
go build -o sandbox/gemma/renvo ./cmd/renvo
sandbox/gemma/renvo cc -t darwin/arm64 -arena-size 1073741824 \
  examples/gemma-c/main.c -o sandbox/gemma/renvo-gemma

# Set this to your downloaded checkpoint directory.
MODEL="$HOME/dev/llm/gemma-3-270m-it"
python3 examples/gemma-c/prepare.py "$MODEL" sandbox/gemma/model.bin
python3 examples/gemma-c/prepare_tokenizer.py "$MODEL" sandbox/gemma/tokenizer.bin

# Subsequent runs need only the executable and the two packed files.
sandbox/gemma/renvo-gemma --chat sandbox/gemma/model.bin sandbox/gemma/tokenizer.bin \
  'What is the capital of France?' 16
```

The packer reads `config.json` and `model.safetensors` from the local checkpoint;
the tokenizer packer reads `tokenizer.json`. Both packers use only Python's
standard library. No weights are downloaded by these scripts. The packed weights
are approximately 511 MiB and tokenizer tables 23.6 MiB; the native runtime
reserves a 1 GiB arena.

## Native tokenizer

`prepare_tokenizer.py` precomputes an open-addressed hash table for all 514,906
BPE merges, a direct Unicode-character lookup, an added-token trie, byte-fallback
IDs, and decoded token strings. `tokenizer.go` reads the tables directly without
building runtime maps or parsing JSON. BPE uses linked token positions and a
min-heap ordered by merge rank and leftmost position, giving O(n log n) merge
processing. Added tokens match before normalization; spaces become `▁` for BPE.
Decoding restores spaces, skips special tokens, and preserves byte-fallback
UTF-8 runs and invalid-run replacement behavior. These semantics follow the
[Hugging Face BPE](https://github.com/huggingface/tokenizers/blob/main/tokenizers/src/models/bpe/word.rs)
and [ByteFallback decoder](https://github.com/huggingface/tokenizers/blob/main/tokenizers/src/decoders/byte_fallback.rs).

The supported chat template is one user message followed by the model turn,
including Python/Jinja-compatible Unicode whitespace trimming. Tokenizer inputs
are limited to 65,536 UTF-8 bytes, and inference remains limited to 1,024 tokens.
The tokenizer supports the checkpoint's extra image special token, but the text
model rejects it before loading weights because it has no corresponding embedding.

Tokenizer-only commands do not load model weights:

```sh
sandbox/gemma/renvo-gemma --encode sandbox/gemma/tokenizer.bin 'Hello world!'
sandbox/gemma/renvo-gemma --tokenize sandbox/gemma/tokenizer.bin 'Hello world!'
sandbox/gemma/renvo-gemma --decode sandbox/gemma/tokenizer.bin output.ids
```

`--encode` emits raw token IDs without BOS or a chat template; `--tokenize` applies
the chat template. Both print one decimal ID per line. `--decode` reads a file of
little-endian uint32 IDs and writes decoded text without an extra newline.

## Reference checks (optional)

```sh
python3.12 -m venv sandbox/llm-venv
sandbox/llm-venv/bin/pip install 'transformers==5.17.0' 'torch==2.14.0' 'numpy==2.5.3' sentencepiece
sandbox/llm-venv/bin/python examples/gemma-c/verify_tokenizer.py \
  --model "$MODEL" --tokenizer sandbox/gemma/tokenizer.bin --runtime sandbox/gemma/renvo-gemma
sandbox/llm-venv/bin/python examples/gemma-c/run.py \
  --model "$MODEL" --weights sandbox/gemma/model.bin \
  --runtime sandbox/gemma/renvo-gemma \
  --prompt 'What is the capital of France?' --steps 16 --verify
```

The native executable passed 168 encoding cases, 27 chat-template cases, and 27
decoding cases in about 22 seconds. These include every added token, every
vocabulary ID in decoding, multilingual text, emoji, repeated whitespace, merge
ties, deterministic random inputs, and valid/invalid byte fallback. Reference
inference requires additional memory for float32 weights. `run.py --verify` uses
the native tokenizer and decoder, comparing both against Hugging Face as well as
checking logits and the entire generated token sequence against PyTorch.

## Scope

This first version supports this model's exact dimensions, BF16 checkpoint,
one sequence, and up to 1,024 tokens including the generation budget. It uses
scalar matrix-vector products without SIMD or threading. It is a correctness
prototype, with no sampling controls, batching, multi-turn chat, or web UI.
Only the macOS arm64 target has been exercised.

Validated on the development Mac: the 16-token prompt “What is the capital of
France?” generated “The capital of France is Paris.” (9 tokens including the
end-of-turn marker) in 35 seconds including loading and prefill. All generated
token IDs matched the float32 PyTorch reference; maximum prompt logit error was
0.000340. This short check exercises cached decoding but does not cover eviction
at the 512-token sliding-window boundary. The compiler regression and repository
preflight also pass.

For that short chat, `/usr/bin/time -l` measured 595 MiB peak resident memory;
tokenizer-only execution used 81 MiB. An isolated instrumented run took 35.3 s:
0.49 s startup/other overhead, 0.112 s tokenizer loading, 0.000085 s tokenization,
1.20 s weight loading, 15.12 s prompt processing, and 18.36 s subsequent decoding.
Matrix products consumed over 99% of inference time (14.38 s feed-forward,
12.86 s vocabulary projection, 6.00 s attention projections). These are individual
development measurements, not performance guarantees or compiler gate thresholds.

`prepare.py` writes a little-endian 2,048-byte header containing dimensions,
absolute tensor offsets, attention settings, and RoPE frequency offsets,
followed by the tensor data. `main.c` validates dimensions and extents before
inference. The native command accepts `model.bin prompt.ids steps [logits.bin]`;
prompt IDs are little-endian uint32 values, output is one token ID per line,
and the optional logit dump contains 262,144 float32 values for the last prompt
position. `run.py --verify` compares these logits and the entire greedy sequence
against the CPU float32 reference.

The example exposed a C frontend arithmetic-conversion bug: mixing a float with
an equally wide or wider integer could incorrectly select integer arithmetic.
The fix is covered by `TestFrontendCMixedFloatArithmetic`.
