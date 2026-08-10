//go:build !ios

package main

import "renvo.dev/forms"

const controlsPlatformSubtitle = "Native Android • 360 dp canvas • 2× density"

func renderTimestampMicroseconds() int { return 0 }

func configureRenderTiming(app *forms.App) {}
