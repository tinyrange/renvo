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

env GOOS=wasip1 GOARCH=wasm go build -trimpath -ldflags='-s -w' \
  -o "$output/renvo-format.wasm" ./cmd/renvowasiformat

env GOOS=wasip1 GOARCH=wasm go build -trimpath -ldflags='-s -w' \
  -o "$output/renvo-backend-jit.wasm" ./cmd/renvowasibackendjit

env GOOS=wasip1 GOARCH=wasm go build -trimpath -ldflags='-s -w' \
  -o "$output/renvo-vm-backend.wasm" ./cmd/renvowasivmbackend

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

# Pure C canonical units currently need the full backend surface. Ordinary Go
# projects keep using the substantially smaller self-hosted module above.
env GOOS=wasip1 GOARCH=wasm go build -trimpath -ldflags='-s -w' \
  -o "$output/backends/native-c.wasm" ./backend

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

build_custom_backend esp32c6-jtag/riscv32 backends/esp32c6_jtag.rtg esp32c6-jtag-riscv32 go
build_custom_backend esp32c6/riscv32 backends/esp32c6.rtg esp32c6-riscv32 go
build_custom_backend esp32s3/xtensa_lx7 backends/esp32s3.rtg esp32s3-xtensa_lx7 go
build_custom_backend esp32p4/riscv32 backends/esp32p4.rtg esp32p4-riscv32 go

build_vm_backend() {
  target_name=$1
  definition=$2
  output_name=$3
  go run ./cmd/renvowasibackendjit \
    -definition "$definition" -target "$target_name" \
    -o "$output/backends/$output_name.rnvb" >/dev/null
}

build_vm_backend msdos/8086 backends/msdos.rtg msdos-8086
build_vm_backend msdos/8086-mz backends/msdos.rtg msdos-8086-mz
cp backends/msdos.rtg "$output/backends/msdos.rtg"

go run ./tools/wasm/cmd/browserassets -o "$output"
cp tools/wasm/browser/index.html tools/wasm/browser/styles.css \
	tools/wasm/browser/app.mjs tools/wasm/browser/worker.mjs \
	tools/wasm/browser/editor-navigation.mjs \
	tools/wasm/browser/language-path.mjs \
	tools/wasm/browser/asset-fetch.mjs \
	tools/wasm/browser/serial-plotter.mjs \
	tools/wasm/browser/project-archive.mjs tools/wasm/browser/device-profile.mjs \
	tools/wasm/browser/c-language.mjs \
	tools/wasm/browser/rtg-language.mjs \
	tools/wasm/browser/test-project.mjs tools/wasm/browser/workspace-store.mjs \
	tools/wasm/browser/service-worker.mjs \
	tools/wasm/browser/esp-webserial.mjs tools/wasm/browser/esp-webusb.mjs \
	tools/wasm/browser/esp-webusb-jtag.mjs "$output/browser/"

if [ "$layout" = pages ]; then
  cp tools/wasm/browser/index.html tools/wasm/browser/styles.css \
    tools/wasm/browser/app.mjs tools/wasm/browser/worker.mjs \
	tools/wasm/browser/editor-navigation.mjs \
	tools/wasm/browser/language-path.mjs \
	tools/wasm/browser/asset-fetch.mjs \
	tools/wasm/browser/serial-plotter.mjs \
	tools/wasm/browser/project-archive.mjs tools/wasm/browser/device-profile.mjs \
	tools/wasm/browser/c-language.mjs \
	tools/wasm/browser/rtg-language.mjs \
	tools/wasm/browser/test-project.mjs tools/wasm/browser/workspace-store.mjs \
	tools/wasm/browser/service-worker.mjs \
    tools/wasm/browser/esp-webserial.mjs tools/wasm/browser/esp-webusb.mjs \
    tools/wasm/browser/esp-webusb-jtag.mjs "$output/"
fi

find "$output" -type f \( -name '*.wasm' -o -name '*.mjs' -o -name '*.js' -o -name '*.css' -o -name '*.json' -o -name '*.html' \) \
  -exec sh -c '
    for file do
      gzip -9 -c "$file" > "$file.gz"
      if command -v brotli >/dev/null 2>&1; then
        brotli -f -q 6 -o "$file.br" "$file"
      fi
    done
  ' sh {} +
