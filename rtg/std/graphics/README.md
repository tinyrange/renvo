# RTG graphics

`graphics` is RTG's portable windowing and software-rendered 2D package. Its
baseline is intentionally small enough to map to modern OpenGL and a future
Win98 GDI/DIB backend without changing application code.

Darwin/arm64 is the first native target. It creates Cocoa windows through
Objective-C runtime calls declared with `rtg:linkstatic`, creates a legacy
OpenGL context, and presents the platform-neutral software surface with
`glDrawPixels`. It uses no cgo or third-party library.

## Coordinate and pixel contract

- The origin is upper-left, positive X points right, and positive Y points down.
- Rectangles are half-open: `[MinX, MaxX) x [MinY, MaxY)`.
- Pixel centres are `(x+0.5, y+0.5)`.
- Application coordinates are `Scalar` (`float64`).
- RGBA8 colors and images use premultiplied alpha.
- `RGBA` accepts straight-alpha channels and premultiplies them.
- `BlendCopy` and premultiplied `BlendSourceOver` are guaranteed.
- `PixelRGBA8` and `PixelA8` are guaranteed image formats.

## Rendering

Surfaces and images are the same CPU-backed resource, so any off-screen surface
can be drawn as an image. Implemented operations include:

- points, filled and stroked rectangles, solid triangles, thick lines and
  polylines;
- convex and general polygons;
- filled and stroked ellipses;
- paths with lines, quadratic and cubic curves, non-zero and even-odd fill
  rules, and software curve flattening;
- nested rectangular clips;
- full 2D affine transforms with push/pop state;
- cropped, scaled, tinted and affine-transformed RGBA8 or A8 images;
- nearest and bilinear sampling, with an integer-aligned scaled-blit fast path;
- sub-image updates and explicit image destruction;
- dirty-region tracking for partial native presentation.

`NewTrueTypeFont` parses dependency-free TrueType `glyf` outlines from a byte
slice at a requested logical pixel height. Glyphs are rasterized lazily into
cached antialiased A8 masks, with Unicode cmap lookup, horizontal metrics,
legacy kern-table kerning, simple quadratic contours, and transformed compound
glyphs. The caller remains responsible for obtaining and licensing font data.

`NewBuiltinFont` remains available as a compact deterministic ASCII fallback.
Text input and measurement use UTF-8; unsupported built-in glyphs render a
replacement character. `DrawGlyphRun` accepts positioned A8 masks for shaped
or externally rasterized glyph runs.

## Windowing

The Darwin backend implements:

- create, close, show, hide, title, client size, repaint requests and present;
- top-down RGBA8 window capture, using `glReadPixels` from the displayed front
  buffer on Darwin (physical pixels on high-DPI displays);
- dependency-free binary PPM export through `Image.EncodePPM`;
- multiple simultaneous windows;
- poll and blocking wait event loops;
- close, resize, focus and expose events;
- pointer move, buttons, wheel, leave and explicit drag capture state;
- physical key up/down, modifiers, repeat state and separate UTF-8 text input;
- cursor selection;
- one-shot timers;
- UTF-8 clipboard text;
- live-resize and backing-scale-aware OpenGL presentation with dirty-row uploads.

Events and drawing coordinates are client-relative logical pixels. Wheel
values are backend-independent scalar deltas rather than Win32 constants.

The ordinary Go (`!rtg`) window implementation is deliberately headless; it
exists to test portable rendering with the Go toolchain. Native windowing is
currently supported only by RTG Darwin/arm64 builds.

## Deliberate baseline exclusions

Shaders, perspective transforms, multisampling, HDR, floating-point targets,
stencil/path clipping, color management, subpixel LCD text, complex blend
equations, layered windows, touch/pen input and identical cross-platform IME
behavior are capabilities outside the guaranteed baseline.
