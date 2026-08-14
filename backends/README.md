# RTG backends

This directory contains complete external backend entrypoints accepted by
`renvo -backend`. Architecture and operating-system fragments shared by those
entrypoints remain in [`backend/definitions`](../backend/definitions), while
frontend target profiles remain in [`systems`](../systems).

The available external backends are:

- `android_arm64.rtg`
- `c89.rtg`
- `esp32c6.rtg`
- `esp32p4.rtg`
- `esp32s3.rtg`
- `ios_arm64.rtg`
