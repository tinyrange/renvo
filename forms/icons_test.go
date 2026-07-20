package forms

import (
	"testing"

	"renvo.dev/std/graphics"
)

func TestEmbeddedIconSetDefinesAndRendersEveryPublicIcon(t *testing.T) {
	loadIconDefinitions()
	if !iconDefinitionsValid {
		t.Fatal("embedded icon set did not parse")
	}
	seen := map[string]bool{}
	for icon := IconNew; icon <= IconControlMenuBar; icon++ {
		name := IconName(icon)
		if name == "" || seen[name] {
			t.Fatalf("icon %d has invalid or duplicate name %q", icon, name)
		}
		seen[name] = true
		if int(icon) >= len(iconDefinitions) || len(iconDefinitions[int(icon)].commands) == 0 {
			t.Fatalf("icon %q has no vector commands", name)
		}

		surface := graphics.NewSurface(24, 24)
		background := graphics.RGBA(250, 251, 253, 255)
		foreground := graphics.RGBA(25, 118, 210, 255)
		surface.FillRect(graphics.R(0, 0, 24, 24), background)
		DrawIcon(surface, icon, graphics.R(4, 4, 16, 16), foreground)
		changed := false
		for y := 0; y < 24 && !changed; y++ {
			for x := 0; x < 24; x++ {
				if surfacePixel(surface, x, y) != background {
					changed = true
					break
				}
			}
		}
		if !changed {
			t.Fatalf("icon %q rendered no pixels", name)
		}
	}
	if len(seen) != IconCount() {
		t.Fatalf("rendered %d icons, want %d", len(seen), IconCount())
	}
}

func TestDesignerControlIconsStayInsideAndCenteredInTheirOpticalBox(t *testing.T) {
	for icon := IconControlLabel; icon <= IconControlMenuBar; icon++ {
		surface := graphics.NewSurface(32, 32)
		DrawControlIcon(surface, icon, graphics.R(4, 4, 24, 24), graphics.White, graphics.RGBA(180, 180, 180, 255), graphics.RGBA(25, 118, 210, 255))
		minX, minY, maxX, maxY := 32, 32, -1, -1
		for y := 0; y < 32; y++ {
			for x := 0; x < 32; x++ {
				if surfacePixel(surface, x, y).A == 0 {
					continue
				}
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
		if maxX < minX || maxY < minY {
			t.Fatalf("control icon %q rendered no pixels", IconName(icon))
		}
		centerX := minX + maxX
		centerY := minY + maxY
		if centerX < 28 || centerX > 34 || centerY < 28 || centerY > 34 {
			t.Fatalf("control icon %q optical bounds (%d,%d)-(%d,%d) are off-center", IconName(icon), minX, minY, maxX, maxY)
		}
		if minX < 3 || minY < 3 || maxX > 28 || maxY > 28 {
			t.Fatalf("control icon %q escaped its padded box: (%d,%d)-(%d,%d)", IconName(icon), minX, minY, maxX, maxY)
		}
	}
}

func TestEmbeddedControlIconMasksContainEveryRasterIcon(t *testing.T) {
	loadControlIconMasks()
	if !controlIconMasksValid {
		t.Fatal("embedded raster control icon masks did not load")
	}
	for icon := 0; icon < controlIconCount; icon++ {
		for layer, mask := range []*graphics.Image{controlIconPrimaryMask, controlIconFillMask, controlIconAccentMask} {
			if mask == nil || mask.Width != controlIconSize*controlIconCount || mask.Height != controlIconSize {
				t.Fatalf("control icon mask layer %d has invalid dimensions", layer)
			}
		}
		visible := false
		for y := 0; y < controlIconSize && !visible; y++ {
			for x := icon * controlIconSize; x < (icon+1)*controlIconSize; x++ {
				if controlIconPrimaryMask.Pixels[y*controlIconPrimaryMask.Stride+x] != 0 || controlIconFillMask.Pixels[y*controlIconFillMask.Stride+x] != 0 || controlIconAccentMask.Pixels[y*controlIconAccentMask.Stride+x] != 0 {
					visible = true
					break
				}
			}
		}
		if !visible {
			t.Fatalf("control icon %q has no embedded raster pixels", IconName(IconControlLabel+Icon(icon)))
		}
	}
}
