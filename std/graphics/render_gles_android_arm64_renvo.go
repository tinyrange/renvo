//go:build renvo && android && arm64

package graphics

// GLES 2 is the portable acceleration floor for Android NativeActivity.

// renvo:linkstatic libEGL.so,eglGetDisplay
func eglGetDisplay(int) int { return 0 }

// renvo:linkstatic libEGL.so,eglInitialize
func eglInitialize(int, *int32, *int32) int { return 0 }

// renvo:linkstatic libEGL.so,eglChooseConfig
func eglChooseConfig(int, *int32, *int, int, *int32) int {
	return 0
}

// renvo:linkstatic libEGL.so,eglGetConfigAttrib
func eglGetConfigAttrib(int, int, int, *int32) int { return 0 }

// renvo:linkstatic libEGL.so,eglCreateWindowSurface
func eglCreateWindowSurface(int, int, int, *int32) int { return 0 }

// renvo:linkstatic libEGL.so,eglCreateContext
func eglCreateContext(int, int, int, *int32) int { return 0 }

// renvo:linkstatic libEGL.so,eglMakeCurrent
func eglMakeCurrent(int, int, int, int) int { return 0 }

// renvo:linkstatic libEGL.so,eglSwapBuffers
func eglSwapBuffers(int, int) int { return 0 }

// renvo:linkstatic libEGL.so,eglDestroyContext
func eglDestroyContext(int, int) int { return 0 }

// renvo:linkstatic libEGL.so,eglDestroySurface
func eglDestroySurface(int, int) int { return 0 }

// renvo:linkstatic libEGL.so,eglTerminate
func eglTerminate(int) int { return 0 }

// renvo:linkstatic libGLESv2.so,glCreateShader
func glCreateShader(int) int { return 0 }

// renvo:linkstatic libGLESv2.so,glShaderSource
func glShaderSource(int, int, **byte, *int32) {}

// renvo:linkstatic libGLESv2.so,glCompileShader
func glCompileShader(int) {}

// renvo:linkstatic libGLESv2.so,glDeleteShader
func glDeleteShader(int) {}

// renvo:linkstatic libGLESv2.so,glCreateProgram
func glCreateProgram() int { return 0 }

// renvo:linkstatic libGLESv2.so,glAttachShader
func glAttachShader(int, int) {}

// renvo:linkstatic libGLESv2.so,glBindAttribLocation
func glBindAttribLocation(int, int, *byte) {}

// renvo:linkstatic libGLESv2.so,glLinkProgram
func glLinkProgram(int) {}

// renvo:linkstatic libGLESv2.so,glDeleteProgram
func glDeleteProgram(int) {}

// renvo:linkstatic libGLESv2.so,glUseProgram
func glUseProgram(int) {}

// renvo:linkstatic libGLESv2.so,glGetUniformLocation
func glGetUniformLocation(int, *byte) int { return -1 }

// renvo:linkstatic libGLESv2.so,glUniform1i
func glUniform1i(int, int) {}

// renvo:linkstatic libGLESv2.so,glUniform2iv
func glUniform2iv(int, int, *int32) {}

// renvo:linkstatic libGLESv2.so,glUniform4iv
func glUniform4iv(int, int, *int32) {}

// renvo:linkstatic libGLESv2.so,glGenBuffers
func glGenBuffers(int, *int32) {}

// renvo:linkstatic libGLESv2.so,glDeleteBuffers
func glDeleteBuffers(int, *int32) {}

// renvo:linkstatic libGLESv2.so,glBindBuffer
func glBindBuffer(int, int) {}

// renvo:linkstatic libGLESv2.so,glBufferData
func glBufferData(int, int, *int32, int) {}

// renvo:linkstatic libGLESv2.so,glEnableVertexAttribArray
func glEnableVertexAttribArray(int) {}

// renvo:linkstatic libGLESv2.so,glVertexAttribPointer
func glVertexAttribPointer(int, int, int, int, int, int) {}

// renvo:linkstatic libGLESv2.so,glViewport
func glViewport(int, int, int, int) {}

// renvo:linkstatic libGLESv2.so,glEnable
func glEnable(int) {}

// renvo:linkstatic libGLESv2.so,glDisable
func glDisable(int) {}

// renvo:linkstatic libGLESv2.so,glBlendFunc
func glBlendFunc(int, int) {}

// renvo:linkstatic libGLESv2.so,glScissor
func glScissor(int, int, int, int) {}

// renvo:linkstatic libGLESv2.so,glGenTextures
func glGenTextures(int, *int32) {}

// renvo:linkstatic libGLESv2.so,glDeleteTextures
func glDeleteTextures(int, *int32) {}

// renvo:linkstatic libGLESv2.so,glBindTexture
func glBindTexture(int, int) {}

// renvo:linkstatic libGLESv2.so,glTexParameteri
func glTexParameteri(int, int, int) {}

// renvo:linkstatic libGLESv2.so,glPixelStorei
func glPixelStorei(int, int) {}

// renvo:linkstatic libGLESv2.so,glTexImage2D
func glTexImage2D(int, int, int, int, int, int, int, int, *byte) {
}

// renvo:linkstatic libGLESv2.so,glDrawArrays
func glDrawArrays(int, int, int) {}

// renvo:linkstatic libGLESv2.so,glReadPixels
func glReadPixels(int, int, int, int, int, int, *byte) {}

const (
	eglNone                 = 0x3038
	eglRedSize              = 0x3024
	eglGreenSize            = 0x3023
	eglBlueSize             = 0x3022
	eglAlphaSize            = 0x3021
	eglSurfaceType          = 0x3033
	eglWindowBit            = 0x0004
	eglRenderableType       = 0x3040
	eglOpenGLES2Bit         = 0x0004
	eglNativeVisualID       = 0x302e
	eglContextClientVersion = 0x3098

	glesTriangles           = 0x0004
	glesOne                 = 1
	glesOneMinusSourceAlpha = 0x0303
	glesBlend               = 0x0be2
	glesScissorTest         = 0x0c11
	glesTexture2D           = 0x0de1
	glesUnsignedByte        = 0x1401
	glesFixed               = 0x140c
	glesRGBA                = 0x1908
	glesAlpha               = 0x1906
	glesTextureMagFilter    = 0x2800
	glesTextureMinFilter    = 0x2801
	glesTextureWrapS        = 0x2802
	glesTextureWrapT        = 0x2803
	glesNearest             = 0x2600
	glesLinear              = 0x2601
	glesClampToEdge         = 0x812f
	glesUnpackAlignment     = 0x0cf5
	glesArrayBuffer         = 0x8892
	glesStreamDraw          = 0x88e0
	glesVertexShader        = 0x8b31
	glesFragmentShader      = 0x8b30
)

type glesTexture struct {
	image    *Image
	texture  int32
	revision int
}

type glesBackend struct {
	display      int
	surface      int
	context      int
	program      int
	vertexBuffer int32
	width        int
	height       int
	uniforms     [5]int
	textures     []glesTexture
	vertices     []int32
}

func glesVertexSource() string {
	return `attribute vec4 aVertex;
uniform ivec2 uSize;
uniform ivec2 uImageSize;
varying vec2 vUV;
void main() {
    vec2 point = aVertex.xy;
    gl_Position = vec4(point.x * 2.0 / float(uSize.x) - 1.0,
                       1.0 - point.y * 2.0 / float(uSize.y), 0.0, 1.0);
    vUV = aVertex.zw / vec2(uImageSize);
}`
}

func glesFragmentSource() string {
	return `precision mediump float;
uniform sampler2D uTexture;
uniform ivec4 uColor;
uniform int uKind;
varying vec2 vUV;
void main() {
    vec4 tint = vec4(uColor) / 255.0;
    if (uKind == 0) {
        gl_FragColor = tint;
        return;
    }
    vec4 sampled = texture2D(uTexture, vUV);
    if (uKind == 2) {
        gl_FragColor = tint * sampled.a;
        return;
    }
    gl_FragColor = sampled * tint;
}`
}

func glesName(name string) []byte { return []byte(name + "\x00") }

func glesCompile(kind int, source string) int {
	shader := glCreateShader(kind)
	if shader == 0 {
		return 0
	}
	bytes := []byte(source + "\x00")
	pointer := &bytes[0]
	glShaderSource(shader, 1, &pointer, nil)
	glCompileShader(shader)
	return shader
}

func (b *glesBackend) initialize(native int) bool {
	b.display = eglGetDisplay(0)
	if b.display == 0 {
		return false
	}
	if eglInitialize(b.display, nil, nil) == 0 {
		return false
	}
	attributes := []int32{
		eglSurfaceType, eglWindowBit,
		eglRenderableType, eglOpenGLES2Bit,
		eglRedSize, 8, eglGreenSize, 8, eglBlueSize, 8, eglAlphaSize, 8,
		eglNone,
	}
	var config int
	var count int32
	if eglChooseConfig(b.display, &attributes[0], &config, 1, &count) == 0 || count < 1 || config == 0 {
		return false
	}
	var visual int32
	if eglGetConfigAttrib(b.display, config, eglNativeVisualID, &visual) == 0 || visual == 0 {
		return false
	}
	if androidSetBuffersGeometry(native, 0, 0, int(visual)) != 0 {
		return false
	}
	b.surface = eglCreateWindowSurface(b.display, config, native, nil)
	if b.surface == 0 {
		return false
	}
	contextAttributes := []int32{eglContextClientVersion, 2, eglNone}
	b.context = eglCreateContext(b.display, config, 0, &contextAttributes[0])
	if b.context == 0 || eglMakeCurrent(b.display, b.surface, b.surface, b.context) == 0 {
		return false
	}
	vertex := glesCompile(glesVertexShader, glesVertexSource())
	fragment := glesCompile(glesFragmentShader, glesFragmentSource())
	b.program = glCreateProgram()
	if vertex == 0 || fragment == 0 || b.program == 0 {
		return false
	}
	glAttachShader(b.program, vertex)
	glAttachShader(b.program, fragment)
	attributeName := glesName("aVertex")
	glBindAttribLocation(b.program, 0, &attributeName[0])
	glLinkProgram(b.program)
	glDeleteShader(vertex)
	glDeleteShader(fragment)
	names := []string{"uSize", "uImageSize", "uColor", "uKind", "uTexture"}
	for i := 0; i < len(names); i++ {
		name := glesName(names[i])
		b.uniforms[i] = glGetUniformLocation(b.program, &name[0])
		if b.uniforms[i] < 0 {
			return false
		}
	}
	glGenBuffers(1, &b.vertexBuffer)
	return b.vertexBuffer != 0
}

func newGLESFrameBackend(window *Window, native int) frameBackend {
	if window == nil || native == 0 {
		return nil
	}
	b := &glesBackend{}
	if !b.initialize(native) {
		b.Destroy()
		return nil
	}
	return b
}

func (b *glesBackend) makeCurrent() bool {
	return b != nil && b.display != 0 && b.surface != 0 && b.context != 0 &&
		eglMakeCurrent(b.display, b.surface, b.surface, b.context) != 0
}

func (b *glesBackend) Begin(width, height int, damage []pixelRect) bool {
	if !b.makeCurrent() || width <= 0 || height <= 0 {
		return false
	}
	if width != b.width || height != b.height {
		if !b.Resize(width, height) {
			return false
		}
	}
	glViewport(0, 0, width, height)
	glUseProgram(b.program)
	viewport := [2]int32{int32(width), int32(height)}
	glUniform2iv(b.uniforms[0], 1, &viewport[0])
	glUniform1i(b.uniforms[4], 0)
	glEnable(glesScissorTest)
	return true
}

func glesPixels(image *Image) *byte {
	if image == nil || len(image.Pixels) == 0 {
		return nil
	}
	return &image.Pixels[0]
}

func (b *glesBackend) textureFor(image *Image, sampling Sampling) int32 {
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
		entry := glesTexture{image: image, revision: -1}
		glGenTextures(1, &entry.texture)
		if entry.texture == 0 {
			return 0
		}
		b.textures = append(b.textures, entry)
		entryIndex = len(b.textures) - 1
	}
	entry := &b.textures[entryIndex]
	glBindTexture(glesTexture2D, int(entry.texture))
	filter := glesNearest
	if sampling == SamplingLinear {
		filter = glesLinear
	}
	glTexParameteri(glesTexture2D, glesTextureMinFilter, filter)
	glTexParameteri(glesTexture2D, glesTextureMagFilter, filter)
	glTexParameteri(glesTexture2D, glesTextureWrapS, glesClampToEdge)
	glTexParameteri(glesTexture2D, glesTextureWrapT, glesClampToEdge)
	if entry.revision != image.revision {
		format := glesRGBA
		if image.Format == PixelA8 {
			format = glesAlpha
		}
		glPixelStorei(glesUnpackAlignment, 1)
		glTexImage2D(glesTexture2D, 0, format, image.Width, image.Height, 0,
			format, glesUnsignedByte, glesPixels(image))
		entry.revision = image.revision
	}
	return entry.texture
}

func glesEncodeScalar(value Scalar) int32 { return int32(value * 65536.0) }

func (b *glesBackend) uploadVertices(list *drawList) bool {
	needed := len(list.vertices) * 4
	if needed == 0 {
		return true
	}
	if needed <= cap(b.vertices) {
		b.vertices = b.vertices[:needed]
	} else {
		b.vertices = make([]int32, needed)
	}
	for i := 0; i < len(list.vertices); i++ {
		vertex := list.vertices[i]
		at := i * 4
		b.vertices[at] = glesEncodeScalar(vertex.x)
		b.vertices[at+1] = glesEncodeScalar(vertex.y)
		b.vertices[at+2] = glesEncodeScalar(vertex.u)
		b.vertices[at+3] = glesEncodeScalar(vertex.v)
	}
	glBindBuffer(glesArrayBuffer, int(b.vertexBuffer))
	glBufferData(glesArrayBuffer, len(b.vertices)*4, &b.vertices[0], glesStreamDraw)
	glEnableVertexAttribArray(0)
	glVertexAttribPointer(0, 4, glesFixed, 0, 16, 0)
	return true
}

func (b *glesBackend) Render(list *drawList) bool {
	if b == nil || list == nil || !b.uploadVertices(list) {
		return false
	}
	for commandIndex := 0; commandIndex < len(list.commands); commandIndex++ {
		command := list.commands[commandIndex]
		glScissor(command.clip.minX, b.height-command.clip.maxY,
			command.clip.maxX-command.clip.minX, command.clip.maxY-command.clip.minY)
		if command.blend == BlendCopy {
			glDisable(glesBlend)
		} else {
			glEnable(glesBlend)
			glBlendFunc(glesOne, glesOneMinusSourceAlpha)
		}
		imageWidth, imageHeight := 1, 1
		if command.kind != drawCommandSolid {
			image := list.images[command.image]
			if b.textureFor(image, command.sampling) == 0 {
				return false
			}
			imageWidth, imageHeight = image.Width, image.Height
		}
		imageSize := [2]int32{int32(imageWidth), int32(imageHeight)}
		color := [4]int32{int32(command.color.R), int32(command.color.G), int32(command.color.B), int32(command.color.A)}
		glUniform2iv(b.uniforms[1], 1, &imageSize[0])
		glUniform4iv(b.uniforms[2], 1, &color[0])
		glUniform1i(b.uniforms[3], int(command.kind))
		glDrawArrays(glesTriangles, command.first, command.count)
	}
	glDisable(glesScissorTest)
	glDisable(glesBlend)
	return true
}

func (b *glesBackend) Present() bool {
	return b != nil && b.display != 0 && b.surface != 0 &&
		eglSwapBuffers(b.display, b.surface) != 0
}

func (b *glesBackend) Resize(width, height int) bool {
	if b == nil || width <= 0 || height <= 0 {
		return false
	}
	b.width, b.height = width, height
	return true
}

func (b *glesBackend) ReadPixels(surface *Surface) bool {
	if b == nil || surface == nil || !b.makeCurrent() || b.width <= 0 || b.height <= 0 {
		return false
	}
	if surface.Width != b.width || surface.Height != b.height {
		surface.Resize(b.width, b.height)
	}
	bottomUp := make([]byte, surface.Stride*surface.Height)
	if len(bottomUp) == 0 {
		return false
	}
	glReadPixels(0, 0, surface.Width, surface.Height, glesRGBA, glesUnsignedByte, &bottomUp[0])
	for y := 0; y < surface.Height; y++ {
		source := (surface.Height - y - 1) * surface.Stride
		destination := y * surface.Stride
		copy(surface.Pixels[destination:destination+surface.Stride],
			bottomUp[source:source+surface.Stride])
	}
	return true
}

func (b *glesBackend) Destroy() {
	if b == nil {
		return
	}
	if b.display != 0 && b.surface != 0 && b.context != 0 {
		eglMakeCurrent(b.display, b.surface, b.surface, b.context)
	}
	if b.vertexBuffer != 0 {
		glDeleteBuffers(1, &b.vertexBuffer)
		b.vertexBuffer = 0
	}
	for i := 0; i < len(b.textures); i++ {
		if b.textures[i].texture != 0 {
			glDeleteTextures(1, &b.textures[i].texture)
		}
	}
	b.textures = nil
	if b.program != 0 {
		glDeleteProgram(b.program)
		b.program = 0
	}
	if b.display != 0 {
		eglMakeCurrent(b.display, 0, 0, 0)
		if b.context != 0 {
			eglDestroyContext(b.display, b.context)
		}
		if b.surface != 0 {
			eglDestroySurface(b.display, b.surface)
		}
		eglTerminate(b.display)
	}
	b.context = 0
	b.surface = 0
	b.display = 0
}

func (b *glesBackend) Stats() RenderStats { return RenderStats{} }
