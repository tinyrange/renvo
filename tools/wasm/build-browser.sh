#!/bin/sh
set -eu

output=${1:-sandbox/wasm/browser}
layout=${2:-nested}
build_dir=sandbox/wasm/build
native=$build_dir/renvo-native
if [ "$(go env GOHOSTOS)" = windows ]; then
  native=$native.exe
fi

case "$layout" in
  nested|pages) ;;
  *)
    echo "usage: $0 [output] [nested|pages]" >&2
    exit 2
    ;;
esac

mkdir -p "$output/backends" "$output/browser"
tools/wasm/build.sh "$output/renvo.wasm" "$output/backends/wasi-wasm32.wasm"

"$native" \
  -tags renvo_wasi_language_service \
  -system systems/frontend-wasi-wasm32.rtg \
  -s -o "$output/renvo-language-service.wasm" ./cmd/renvowasilanguageservice

backend_files=$(go list -f '{{range .GoFiles}}backend/{{.}} {{end}}' ./backend)

stage_backend() {
  destination=$1
  mode=$2
  mkdir -p "$destination"
  staged=""
  for source in $backend_files; do
    name=$(basename "$source")
    if [ "$mode" != go ] && [ "$name" = "renvo_main.go" ]; then
      continue
    fi
    if [ "$mode" != builtin ] && { [ "$name" = "compiler_rtg_generated_impl.go" ] || [ "$name" = "compiler_rtg_inactive_impl.go" ]; }; then
      continue
    fi
    target=$destination/$name
    cp "$source" "$target"
    staged="$staged $target"
  done
  printf '%s' "$staged"
}

native_source_dir=$build_dir/browser-native-backend
native_sources=$(stage_backend "$native_source_dir" builtin)
# shellcheck disable=SC2086 # repository-owned staged paths contain no spaces.
"$native" \
  -system systems/backend-wasi-wasm32.rtg \
  -s -o "$output/backends/native.wasm" $native_sources

build_custom_backend() {
  target_name=$1
  definition=$2
  output_name=$3
  compiler=${4:-renvo}
  source_dir=$build_dir/browser-$output_name-source
  generated=$source_dir/compiler_rtg_prepared_impl.go
  custom_sources=$(stage_backend "$source_dir" "$compiler")
	go run ./internal/backendcompiled/cmd/gen \
	  -prepare-source "$source_dir/compiler_target_policy_impl.go" \
	  -o "$source_dir/compiler_target_policy_impl.go"
  go run ./internal/rtg/cmd/rtggen \
    -prepared -t "$target_name" -o "$generated" "$definition"
  if [ "$compiler" = go ]; then
    # shellcheck disable=SC2086 # repository-owned staged paths contain no spaces.
    env GOOS=wasip1 GOARCH=wasm go build -trimpath -ldflags='-s -w' \
      -o "$output/backends/$output_name.wasm" $custom_sources "$generated"
  else
    # shellcheck disable=SC2086 # repository-owned staged paths contain no spaces.
    "$native" \
      -system systems/backend-wasi-wasm32.rtg \
      -s -o "$output/backends/$output_name.wasm" $custom_sources "$generated"
  fi
}

build_custom_backend esp32c6/riscv32 examples/m5nanoc6/esp32c6.rtg esp32c6-riscv32 go
build_custom_backend esp32s3/xtensa_lx7 examples/m5sticks3/esp32s3.rtg esp32s3-xtensa_lx7 go
build_custom_backend esp32p4/riscv32 examples/m5tab5/esp32p4.rtg esp32p4-riscv32 go

go run ./tools/wasm/cmd/browserassets -o "$output"
cp tools/wasm/browser/index.html tools/wasm/browser/styles.css \
	tools/wasm/browser/app.mjs tools/wasm/browser/worker.mjs \
	tools/wasm/browser/esp-webserial.mjs tools/wasm/browser/esp-webusb.mjs "$output/browser/"

if [ "$layout" = pages ]; then
  cp tools/wasm/browser/index.html tools/wasm/browser/styles.css \
    tools/wasm/browser/app.mjs tools/wasm/browser/worker.mjs \
    tools/wasm/browser/esp-webserial.mjs tools/wasm/browser/esp-webusb.mjs "$output/"
fi
