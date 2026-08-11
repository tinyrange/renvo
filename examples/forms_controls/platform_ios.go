//go:build ios

package main

import "renvo.dev/std/graphics"

const controlsPlatformSubtitle = "Native iOS • Metal accelerated Forms"

func controlsRenderer() graphics.Renderer { return graphics.RendererMetal }
