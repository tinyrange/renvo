//go:build !ios && (!renvo || !arm64 || (!darwin && !android))

package main

import (
	"renvo.dev/forms"
	"renvo.dev/std/graphics"
)

const controlsPlatformSubtitle = "Native Android • 360 dp canvas • 2× density"

func controlsRenderer() graphics.Renderer { return graphics.RendererAuto }

func renderTimestampMicroseconds() int { return 0 }

func configureRenderTiming(app *forms.App) {}
