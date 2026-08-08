#!/bin/sh
set -eu

output=${1:-sandbox/wasm/renvowasi.wasm}
backend_output=${2:-${output%.wasm}-backend.wasm}
build_dir=sandbox/wasm/build

mkdir -p "$build_dir" "$(dirname "$output")" "$(dirname "$backend_output")"
go build -o "$build_dir/renvo-backend" ./backend
go build -tags renvo_bundle -o "$build_dir/renvo-bootstrap" ./cmd/renvobootstrap
"$build_dir/renvo-bootstrap" \
  -tags renvo_bundle \
  -system systems/frontend-linux-amd64.rtg \
  -s -o "$build_dir/renvo-native" ./cmd/renvo
"$build_dir/renvo-native" \
  -tags renvo_wasi_frontend \
  -system systems/frontend-wasi-wasm32.rtg \
  -s -o "$output" ./cmd/renvowasi

# Stage the active ordinary-Go backend package beside the fixed-module
# entrypoint. Renvo intentionally requires explicit source files to share a
# directory; the staged package also keeps the production module independent of
# Go module loading at runtime.
backend_source_dir=$build_dir/wasi-backend-source
mkdir -p "$backend_source_dir"
backend_files=$(go list -f '{{range .GoFiles}}{{if ne . "compiler_main.go"}}{{if ne . "renvo_main.go"}}backend/{{.}} {{end}}{{end}}{{end}}' ./backend)
staged_backend_files=""
for source in $backend_files; do
  destination=$backend_source_dir/$(basename "$source")
  cp "$source" "$destination"
  staged_backend_files="$staged_backend_files $destination"
done
backend_entry=$backend_source_dir/compiler_wasi_module_main_impl.go
cp cmd/renvowasibackend/main_renvo.go "$backend_entry"
# shellcheck disable=SC2086 # staged filenames are repository-owned and space-free.
"$build_dir/renvo-native" \
  -tags renvo_wasi_backend \
  -system systems/backend-wasi-wasm32.rtg \
  -s -o "$backend_output" $staged_backend_files "$backend_entry"
