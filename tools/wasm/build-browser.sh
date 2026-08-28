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

mkdir -p "$output/backends/definitions" "$output/browser/firmware"
tools/wasm/build.sh "$output/renvo.wasm" "$output/backends/wasi-wasm32.wasm"

"$native" \
  -tags renvo_wasi_linker \
  -system systems/frontend-wasi-wasm32.rtg \
  -s -o "$output/renvo-linker.wasm" ./cmd/renvowasilinker

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
build_custom_backend rp2/thumb backends/rp2.rtg rp2-thumb go
build_custom_backend rp2-debug/thumb backends/rp2_debug.rtg rp2-debug-thumb go
build_custom_backend rp2350/riscv32 backends/rp2350.rtg rp2350-riscv32 go

# The resident Pico monitor is itself a Renvo program. No vendor SDK or native
# embedded toolchain participates in the browser artifact.
go run ./cmd/renvo -backend backends/rp2.rtg -t rp2/thumb -tags pico2 -arena-size 8192 \
  -o "$output/browser/firmware/renvo-rp2-monitor.uf2" ./cmd/renvopico-monitor

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
build_vm_backend bios/8086 backends/bios_multistage.rtg bios-8086
build_vm_backend uefi/amd64 backends/uefi_amd64.rtg uefi-amd64
build_vm_backend unixv7/pdp11 examples/pdp11v7/pdp11_v7.rbe unixv7-pdp11
cp backends/msdos.rtg "$output/backends/msdos.rtg"
cp backends/bios_8086.rtg "$output/backends/bios_8086.rtg"
cp backends/bios_multistage.rtg "$output/backends/bios_multistage.rtg"
cp backends/freestanding_amd64.rtg "$output/backends/freestanding_amd64.rtg"
cp backends/uefi_amd64.rtg "$output/backends/uefi_amd64.rtg"
mkdir -p "$output/examples/pdp11v7"
cp examples/pdp11v7/pdp11_v7.rbe "$output/examples/pdp11v7/pdp11_v7.rbe"
cp backend/definitions/x86_64.rtg backend/definitions/elf_amd64_primitives.rtg \
	"$output/backends/definitions/"

go run ./tools/wasm/cmd/browserassets -o "$output"
cp tools/wasm/browser/index.html tools/wasm/browser/styles.css \
	tools/wasm/browser/app.mjs tools/wasm/browser/worker.mjs \
	tools/wasm/browser/build-readiness.mjs \
	tools/wasm/browser/target-capabilities.mjs \
	tools/wasm/browser/api-help.mjs \
	tools/wasm/browser/editor-navigation.mjs \
	tools/wasm/browser/language-path.mjs \
	tools/wasm/browser/asset-fetch.mjs \
	tools/wasm/browser/serial-plotter.mjs \
	tools/wasm/browser/project-archive.mjs tools/wasm/browser/device-profile.mjs \
	tools/wasm/browser/c-language.mjs \
	tools/wasm/browser/makefile-language.mjs tools/wasm/browser/makefile.mjs \
	tools/wasm/browser/rtg-language.mjs \
	tools/wasm/browser/test-project.mjs tools/wasm/browser/workspace-store.mjs \
	tools/wasm/browser/service-worker.mjs \
	tools/wasm/browser/esp-webserial.mjs tools/wasm/browser/esp-webusb.mjs \
	tools/wasm/browser/esp-webusb-jtag.mjs tools/wasm/browser/pico-cmsis-dap.mjs \
	tools/wasm/browser/pico-webusb-monitor.mjs "$output/browser/"

if [ "$layout" = pages ]; then
  mkdir -p "$output/firmware"
  cp "$output/browser/firmware/renvo-rp2-monitor.uf2" "$output/firmware/"
  cp tools/wasm/browser/index.html tools/wasm/browser/styles.css \
    tools/wasm/browser/app.mjs tools/wasm/browser/worker.mjs \
	tools/wasm/browser/build-readiness.mjs \
	tools/wasm/browser/target-capabilities.mjs \
	tools/wasm/browser/api-help.mjs \
	tools/wasm/browser/editor-navigation.mjs \
	tools/wasm/browser/language-path.mjs \
	tools/wasm/browser/asset-fetch.mjs \
	tools/wasm/browser/serial-plotter.mjs \
	tools/wasm/browser/project-archive.mjs tools/wasm/browser/device-profile.mjs \
	tools/wasm/browser/c-language.mjs \
	tools/wasm/browser/makefile-language.mjs tools/wasm/browser/makefile.mjs \
	tools/wasm/browser/rtg-language.mjs \
	tools/wasm/browser/test-project.mjs tools/wasm/browser/workspace-store.mjs \
	tools/wasm/browser/service-worker.mjs \
    tools/wasm/browser/esp-webserial.mjs tools/wasm/browser/esp-webusb.mjs \
    tools/wasm/browser/esp-webusb-jtag.mjs tools/wasm/browser/pico-cmsis-dap.mjs \
    tools/wasm/browser/pico-webusb-monitor.mjs "$output/"
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
