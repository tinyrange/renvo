//go:build renvo && darwin && !ios && arm64

package main

import "renvo.dev/std/graphics"

const controlsPlatformSubtitle = "Native macOS • Metal accelerated Forms"

func controlsRenderer() graphics.Renderer { return graphics.RendererAuto }
