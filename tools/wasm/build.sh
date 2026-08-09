#!/bin/sh
set -eu

output=${1:-sandbox/wasm/renvowasi.wasm}
backend_output=${2:-${output%.wasm}-backend.wasm}
build_dir=sandbox/wasm/build
host_os=$(go env GOHOSTOS)
host_arch=$(go env GOHOSTARCH)
executable_suffix=
if [ "$host_os" = windows ]; then
	executable_suffix=.exe
fi
backend=$build_dir/renvo-backend$executable_suffix
bootstrap=$build_dir/renvo-bootstrap$executable_suffix
native=$build_dir/renvo-native$executable_suffix

case "$host_os/$host_arch" in
	linux/amd64) host_target=linux/amd64 ;;
	linux/386) host_target=linux/386 ;;
	linux/arm64) host_target=linux/aarch64 ;;
	linux/arm) host_target=linux/arm ;;
	darwin/arm64) host_target=darwin/arm64 ;;
	windows/amd64) host_target=windows/amd64 ;;
	windows/386) host_target=windows/386 ;;
	windows/arm64) host_target=windows/arm64 ;;
	*)
		echo "tools/wasm/build.sh: unsupported build host $host_os/$host_arch" >&2
		exit 1
		;;
esac

mkdir -p "$build_dir" "$(dirname "$output")" "$(dirname "$backend_output")"
env GOOS="$host_os" GOARCH="$host_arch" go build -o "$backend" ./backend
env GOOS="$host_os" GOARCH="$host_arch" go build -tags renvo_bundle -o "$bootstrap" ./cmd/renvobootstrap
"$bootstrap" \
  -tags renvo_bundle \
  -t "$host_target" -arena-size 134217728 \
  -s -o "$native" ./cmd/renvo
"$native" \
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
"$native" \
  -tags renvo_wasi_backend \
  -system systems/backend-wasi-wasm32.rtg \
  -s -o "$backend_output" $staged_backend_files "$backend_entry"
