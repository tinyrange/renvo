//go:build renvo && android && arm64

package main

import (
	"renvo.dev/forms"
	"renvo.dev/std/graphics"
)

const controlsPlatformSubtitle = "Native Android • OpenGL ES accelerated Forms"

func controlsRenderer() graphics.Renderer { return graphics.RendererOpenGL }

func renderTimestampMicroseconds() int { return 0 }

func configureRenderTiming(app *forms.App) {}
