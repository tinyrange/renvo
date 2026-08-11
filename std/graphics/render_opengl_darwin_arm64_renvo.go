//go:build renvo && darwin && !ios && arm64

package graphics

// The macOS OpenGL backend uses the legacy context already owned by Window.
// Geometry, sampling, blending, and framebuffer writes execute on the GPU; a
// persistent texture preserves pixels outside Forms damage regions.

// renvo:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glEnable
func glEnable(capability int) {}

// renvo:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glDisable
func glDisable(capability int) {}

// renvo:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glBlendFunc
func glBlendFunc(source, destination int) {}

// renvo:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glScissor
func glScissor(x, y, width, height int) {}

// renvo:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glBegin
func glBegin(mode int) {}

// renvo:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glEnd
func glEnd() {}

// renvo:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glColor4ub
func glColor4ub(red, green, blue, alpha int) {}

// renvo:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glVertex2d
func glVertex2d(x, y Scalar) {}

// renvo:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glTexCoord2d
func glTexCoord2d(s, t Scalar) {}

// renvo:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glGenTextures
func glGenTextures(count int, textures *int32) {}

// renvo:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glDeleteTextures
func glDeleteTextures(count int, textures *int32) {}

// renvo:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glBindTexture
func glBindTexture(target, texture int) {}

// renvo:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glTexParameteri
func glTexParameteri(target, name, value int) {}

// renvo:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glTexImage2D
func glTexImage2D(target, level, internalFormat, width, height, border, format, typ int, pixels *byte) {
}

// renvo:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glCopyTexSubImage2D
func glCopyTexSubImage2D(target, level, xOffset, yOffset, x, y, width, height int) {}

const (
	glTriangles           = 0x0004
	glQuads               = 0x0007
	glBack                = 0x0405
	glBlend               = 0x0be2
	glScissorTest         = 0x0c11
	glTexture2D           = 0x0de1
	glTextureMagFilter    = 0x2800
	glTextureMinFilter    = 0x2801
	glTextureWrapS        = 0x2802
	glTextureWrapT        = 0x2803
	glClampToEdge         = 0x812f
	glNearest             = 0x2600
	glLinear              = 0x2601
	glOne                 = 1
	glOneMinusSourceAlpha = 0x0303
	glAlpha               = 0x1906
)

type openGLTextureEntry struct {
	image    *Image
	texture  int32
	width    int
	height   int
	format   PixelFormat
	revision int
}

type openGLFrameBackend struct {
	window         *Window
	width          int
	height         int
	backingTexture int32
	backingValid   bool
	textures       []openGLTextureEntry
	stats          RenderStats
	renderStarted  Scalar
}

func newOpenGLFrameBackend(window *Window) frameBackend {
	if window == nil || window.context == 0 {
		return nil
	}
	return &openGLFrameBackend{window: window}
}

func (b *openGLFrameBackend) Begin(width, height int, damage []pixelRect) bool {
	if b == nil || b.window == nil || b.window.context == 0 || width <= 0 || height <= 0 {
		return false
	}
	objcMsg0(b.window.context, selector("makeCurrentContext"))
	if width != b.width || height != b.height || b.backingTexture == 0 {
		if !b.Resize(width, height) {
			return false
		}
	}
	b.renderStarted = darwinNow()
	glViewport(0, 0, width, height)
	glMatrixMode(glProjection)
	glLoadIdentity()
	glOrtho(0, width, height, 0, -1, 1)
	glMatrixMode(glModelView)
	glLoadIdentity()
	glDrawBuffer(glBack)
	glDisable(glScissorTest)
	glDisable(glBlend)
	if b.backingValid {
		b.drawBackingTexture()
	}
	return true
}

func (b *openGLFrameBackend) drawBackingTexture() {
	glEnable(glTexture2D)
	glBindTexture(glTexture2D, int(b.backingTexture))
	glTexParameteri(glTexture2D, glTextureMinFilter, glNearest)
	glTexParameteri(glTexture2D, glTextureMagFilter, glNearest)
	glColor4ub(255, 255, 255, 255)
	glBegin(glQuads)
	// glCopyTexSubImage2D stores the framebuffer's bottom row at texture v=0.
	glTexCoord2d(0, 1)
	glVertex2d(0, 0)
	glTexCoord2d(1, 1)
	glVertex2d(Scalar(b.width), 0)
	glTexCoord2d(1, 0)
	glVertex2d(Scalar(b.width), Scalar(b.height))
	glTexCoord2d(0, 0)
	glVertex2d(0, Scalar(b.height))
	glEnd()
	glDisable(glTexture2D)
}

func (b *openGLFrameBackend) configureCommand(command drawCommand) {
	glEnable(glScissorTest)
	glScissor(command.clip.minX, b.height-command.clip.maxY, command.clip.maxX-command.clip.minX, command.clip.maxY-command.clip.minY)
	if command.blend == BlendCopy {
		glDisable(glBlend)
	} else {
		glEnable(glBlend)
		glBlendFunc(glOne, glOneMinusSourceAlpha)
	}
	glColor4ub(int(command.color.R), int(command.color.G), int(command.color.B), int(command.color.A))
}

func imagePixels(image *Image) *byte {
	if image == nil || len(image.Pixels) == 0 {
		return nil
	}
	return &image.Pixels[0]
}

func (b *openGLFrameBackend) textureFor(image *Image, sampling Sampling) int32 {
	if image == nil || image.Width <= 0 || image.Height <= 0 {
		return 0
	}
	entryIndex := -1
	for i := 0; i < len(b.textures); i++ {
		if b.textures[i].image == image {
			entryIndex = i
			break
		}
	}
	if entryIndex < 0 {
		entry := openGLTextureEntry{image: image}
		glGenTextures(1, &entry.texture)
		if entry.texture == 0 {
			return 0
		}
		b.textures = append(b.textures, entry)
		entryIndex = len(b.textures) - 1
	}
	entry := &b.textures[entryIndex]
	glBindTexture(glTexture2D, int(entry.texture))
	filter := glNearest
	if sampling == SamplingLinear {
		filter = glLinear
	}
	glTexParameteri(glTexture2D, glTextureMinFilter, filter)
	glTexParameteri(glTexture2D, glTextureMagFilter, filter)
	glTexParameteri(glTexture2D, glTextureWrapS, glClampToEdge)
	glTexParameteri(glTexture2D, glTextureWrapT, glClampToEdge)
	if entry.width != image.Width || entry.height != image.Height || entry.format != image.Format || entry.revision != image.revision {
		format := glRGBA
		if image.Format == PixelA8 {
			format = glAlpha
		}
		glPixelStorei(glUnpackAlignment, 1)
		glPixelStorei(glUnpackRowLength, 0)
		glTexImage2D(glTexture2D, 0, format, image.Width, image.Height, 0, format, glUnsignedByte, imagePixels(image))
		entry.width = image.Width
		entry.height = image.Height
		entry.format = image.Format
		entry.revision = image.revision
		b.stats.TextureBytes += image.Stride * image.Height
	}
	return entry.texture
}

func (b *openGLFrameBackend) Render(list *drawList) bool {
	if b == nil || list == nil {
		return false
	}
	b.stats.TextureBytes = 0
	for commandIndex := 0; commandIndex < len(list.commands); commandIndex++ {
		command := list.commands[commandIndex]
		if command.count <= 0 || command.first < 0 || command.first+command.count > len(list.vertices) {
			continue
		}
		b.configureCommand(command)
		var image *Image
		if command.kind == drawCommandSolid {
			glDisable(glTexture2D)
		} else {
			if command.image < 0 || command.image >= len(list.images) {
				continue
			}
			image = list.images[command.image]
			if b.textureFor(image, command.sampling) == 0 {
				return false
			}
			glEnable(glTexture2D)
		}
		glBegin(glTriangles)
		for vertexIndex := command.first; vertexIndex < command.first+command.count; vertexIndex++ {
			vertex := list.vertices[vertexIndex]
			if image != nil {
				glTexCoord2d(vertex.u/Scalar(image.Width), vertex.v/Scalar(image.Height))
			}
			glVertex2d(vertex.x, vertex.y)
		}
		glEnd()
	}
	glDisable(glScissorTest)
	glDisable(glBlend)
	glDisable(glTexture2D)
	glBindTexture(glTexture2D, int(b.backingTexture))
	glCopyTexSubImage2D(glTexture2D, 0, 0, 0, 0, 0, b.width, b.height)
	b.backingValid = true
	b.stats.SubmitSeconds = darwinNow() - b.renderStarted
	return true
}

func (b *openGLFrameBackend) Present() bool {
	if b == nil || b.window == nil || b.window.context == 0 {
		return false
	}
	started := darwinNow()
	glFlush()
	objcMsg0(b.window.context, selector("flushBuffer"))
	b.stats.PresentSeconds = darwinNow() - started
	return true
}

func (b *openGLFrameBackend) Resize(width, height int) bool {
	if b == nil || width <= 0 || height <= 0 {
		return false
	}
	if b.backingTexture == 0 {
		glGenTextures(1, &b.backingTexture)
	}
	if b.backingTexture == 0 {
		return false
	}
	b.width, b.height = width, height
	glBindTexture(glTexture2D, int(b.backingTexture))
	glTexParameteri(glTexture2D, glTextureMinFilter, glNearest)
	glTexParameteri(glTexture2D, glTextureMagFilter, glNearest)
	glTexParameteri(glTexture2D, glTextureWrapS, glClampToEdge)
	glTexParameteri(glTexture2D, glTextureWrapT, glClampToEdge)
	glTexImage2D(glTexture2D, 0, glRGBA, width, height, 0, glRGBA, glUnsignedByte, nil)
	b.backingValid = false
	return true
}

func (b *openGLFrameBackend) ReadPixels(surface *Surface) bool {
	if b == nil || b.window == nil || b.window.context == 0 || surface == nil {
		return false
	}
	width, height := surface.Width, surface.Height
	if b.width > 0 && b.height > 0 {
		width, height = b.width, b.height
		if surface.Width != width || surface.Height != height {
			surface.Resize(width, height)
		}
	}
	objcMsg0(b.window.context, selector("makeCurrentContext"))
	glFinish()
	glReadBuffer(glFront)
	glPixelStorei(glPackAlignment, 1)
	bottomUp := make([]byte, surface.Stride*height)
	glReadPixels(0, 0, width, height, glRGBA, glUnsignedByte, bottomUp)
	row := surface.Stride
	for y := 0; y < height; y++ {
		copy(surface.Pixels[y*row:(y+1)*row], bottomUp[(height-y-1)*row:(height-y)*row])
	}
	return true
}

func (b *openGLFrameBackend) Destroy() {
	if b == nil {
		return
	}
	if b.window != nil && b.window.context != 0 {
		objcMsg0(b.window.context, selector("makeCurrentContext"))
	}
	if b.backingTexture != 0 {
		glDeleteTextures(1, &b.backingTexture)
		b.backingTexture = 0
	}
	for i := 0; i < len(b.textures); i++ {
		if b.textures[i].texture != 0 {
			glDeleteTextures(1, &b.textures[i].texture)
		}
	}
	b.textures = nil
	b.backingValid = false
}

func (b *openGLFrameBackend) Stats() RenderStats {
	if b == nil {
		return RenderStats{}
	}
	return b.stats
}
