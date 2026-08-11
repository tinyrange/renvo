//go:build renvo && darwin && arm64

package graphics

const (
	metalPixelFormatBGRA8               = 80
	metalLoadActionDontCare             = 0
	metalStoreActionStore               = 1
	metalPrimitiveTriangle              = 3
	metalBlendFactorOne                 = 1
	metalBlendFactorOneMinusSourceAlpha = 5
	metalBlendOperationAdd              = 0
	metalResourceStorageShared          = 0
	metalCommandBytes                   = 64
)

type metalRegion struct {
	originX int
	originY int
	originZ int
	sizeX   int
	sizeY   int
	sizeZ   int
}

type metalImageEntry struct {
	image    *Image
	buffer   int
	revision int
}

type metalFrameBackend struct {
	window         *Window
	device         int
	queue          int
	layer          int
	library        int
	copyPipeline   int
	blendPipeline  int
	width          int
	height         int
	drawable       int
	command        int
	vertexBuffer   int
	frameBuffer    int
	commandBuffer  int
	dummyBuffer    int
	lastTexture    int
	uniformBuffers []int
	images         []metalImageEntry
	stats          RenderStats
	renderStarted  Scalar
}

func metalShaderSource() string {
	return `#include <metal_stdlib>
using namespace metal;

struct DrawVertex { int4 value; };
struct VertexOut {
    float4 position [[position]];
    float2 uv;
};
struct DrawInfo {
    int4 color;
    int4 image;
    int4 clip;
    int4 extra;
};

vertex VertexOut renvoVertex(device const DrawVertex *vertices [[buffer(0)]],
                             constant int2 &size [[buffer(1)]],
                             uint id [[vertex_id]]) {
    float2 point = float2(vertices[id].value.xy) / 256.0;
    VertexOut out;
    out.position = float4(point.x * 2.0 / float(size.x) - 1.0,
                          1.0 - point.y * 2.0 / float(size.y), 0.0, 1.0);
    out.uv = float2(vertices[id].value.zw) / 256.0;
    return out;
}

float4 renvoPixel(device const uchar *pixels, int stride, int x, int y) {
    int at = y * stride + x * 4;
    return float4(pixels[at], pixels[at + 1], pixels[at + 2], pixels[at + 3]) / 255.0;
}

fragment float4 renvoFragment(VertexOut in [[stage_in]],
                              constant DrawInfo &info [[buffer(0)]],
                              device const uchar *pixels [[buffer(1)]]) {
    int2 position = int2(in.position.xy);
    if (position.x < info.clip.x || position.y < info.clip.y ||
        position.x >= info.clip.z || position.y >= info.clip.w) {
        discard_fragment();
    }
    float4 tint = float4(info.color) / 255.0;
    int kind = info.image.x;
    if (kind == 0) return tint;
    int width = info.image.y;
    int height = info.image.z;
    int stride = info.image.w;
    float2 uv = in.uv;
    int x = clamp(int(floor(uv.x)), 0, width - 1);
    int y = clamp(int(floor(uv.y)), 0, height - 1);
    if (kind == 2) {
        float alpha = float(pixels[y * stride + x]) / 255.0;
        return tint * alpha;
    }
    float4 sampled;
    if (info.extra.x == 0) {
        sampled = renvoPixel(pixels, stride, x, y);
    } else {
        float2 samplePoint = uv - 0.5;
        int2 low = int2(floor(samplePoint));
        float2 amount = fract(samplePoint);
        int x0 = clamp(low.x, 0, width - 1);
        int y0 = clamp(low.y, 0, height - 1);
        int x1 = clamp(low.x + 1, 0, width - 1);
        int y1 = clamp(low.y + 1, 0, height - 1);
        float4 top = mix(renvoPixel(pixels, stride, x0, y0), renvoPixel(pixels, stride, x1, y0), amount.x);
        float4 bottom = mix(renvoPixel(pixels, stride, x0, y1), renvoPixel(pixels, stride, x1, y1), amount.x);
        sampled = mix(top, bottom, amount.y);
    }
    return sampled * tint;
}`
}

func newMetalFrameBackend(window *Window) frameBackend {
	if window == nil || window.view == 0 {
		return nil
	}
	b := &metalFrameBackend{window: window}
	b.device = metalCreateSystemDefaultDevice()
	if b.device == 0 {
		return nil
	}
	b.queue = objcMsg0(b.device, selector("newCommandQueue"))
	if b.queue == 0 {
		b.Destroy()
		return nil
	}
	source := cocoaString(metalShaderSource())
	b.library = objcMsg3(b.device, selector("newLibraryWithSource:options:error:"), source, 0, 0)
	if b.library == 0 {
		b.Destroy()
		return nil
	}
	vertex := objcMsg1(b.library, selector("newFunctionWithName:"), cocoaString("renvoVertex"))
	fragment := objcMsg1(b.library, selector("newFunctionWithName:"), cocoaString("renvoFragment"))
	if vertex == 0 || fragment == 0 {
		b.Destroy()
		return nil
	}
	b.copyPipeline = b.newPipeline(vertex, fragment, false)
	b.blendPipeline = b.newPipeline(vertex, fragment, true)
	objcMsg0(vertex, selector("release"))
	objcMsg0(fragment, selector("release"))
	if b.copyPipeline == 0 || b.blendPipeline == 0 {
		b.Destroy()
		return nil
	}
	b.layer = objcMsg0(objcGetClass("CAMetalLayer"), selector("layer"))
	if b.layer == 0 {
		b.Destroy()
		return nil
	}
	objcMsg0(b.layer, selector("retain"))
	objcMsg1(b.layer, selector("setDevice:"), b.device)
	objcMsg1(b.layer, selector("setPixelFormat:"), metalPixelFormatBGRA8)
	objcMsg1(b.layer, selector("setFramebufferOnly:"), 0)
	if !metalAttachLayer(window, b.layer) {
		b.Destroy()
		return nil
	}
	dummy := []byte{0, 0, 0, 0}
	b.dummyBuffer = objcMsgBytes4(b.device, selector("newBufferWithBytes:length:options:"), dummy, len(dummy), metalResourceStorageShared)
	if b.dummyBuffer == 0 {
		b.Destroy()
		return nil
	}
	return b
}

func (b *metalFrameBackend) newPipeline(vertex, fragment int, blending bool) int {
	descriptor := objcMsg0(objcGetClass("MTLRenderPipelineDescriptor"), selector("alloc"))
	descriptor = objcMsg0(descriptor, selector("init"))
	if descriptor == 0 {
		return 0
	}
	objcMsg1(descriptor, selector("setVertexFunction:"), vertex)
	objcMsg1(descriptor, selector("setFragmentFunction:"), fragment)
	attachments := objcMsg0(descriptor, selector("colorAttachments"))
	attachment := objcMsg1(attachments, selector("objectAtIndexedSubscript:"), 0)
	objcMsg1(attachment, selector("setPixelFormat:"), metalPixelFormatBGRA8)
	if blending {
		objcMsg1(attachment, selector("setBlendingEnabled:"), 1)
		objcMsg1(attachment, selector("setRgbBlendOperation:"), metalBlendOperationAdd)
		objcMsg1(attachment, selector("setAlphaBlendOperation:"), metalBlendOperationAdd)
		objcMsg1(attachment, selector("setSourceRGBBlendFactor:"), metalBlendFactorOne)
		objcMsg1(attachment, selector("setDestinationRGBBlendFactor:"), metalBlendFactorOneMinusSourceAlpha)
		objcMsg1(attachment, selector("setSourceAlphaBlendFactor:"), metalBlendFactorOne)
		objcMsg1(attachment, selector("setDestinationAlphaBlendFactor:"), metalBlendFactorOneMinusSourceAlpha)
	}
	pipeline := objcMsg2(b.device, selector("newRenderPipelineStateWithDescriptor:error:"), descriptor, 0)
	objcMsg0(descriptor, selector("release"))
	return pipeline
}

func metalAppendInt32(buffer []byte, value int) []byte {
	return append(buffer, byte(value), byte(value>>8), byte(value>>16), byte(value>>24))
}

func metalRelease(value int) {
	if value != 0 {
		objcMsg0(value, selector("release"))
	}
}

// Objective-C sees a byte slice as a single pointer. Keep the slice expansion
// on the Go side so its hidden length and capacity do not shift native method
// arguments into the wrong registers.
func objcMsgBytes4(object, selector int, value []byte, length, options int) int {
	if len(value) == 0 || length <= 0 {
		return 0
	}
	if length > len(value) {
		length = len(value)
	}
	return objcMsgPointer3(object, selector, &value[0], length, options)
}

func metalGetBytes(object, selector int, bytes []byte, bytesPerRow int, region *metalRegion, level int) int {
	if len(bytes) == 0 {
		return 0
	}
	return metalGetBytesRaw(object, selector, &bytes[0], bytesPerRow, region, level)
}

func (b *metalFrameBackend) Begin(width, height int, damage []pixelRect) bool {
	if b == nil || b.layer == 0 || width <= 0 || height <= 0 {
		return false
	}
	if width != b.width || height != b.height {
		return b.Resize(width, height)
	}
	return true
}

func (b *metalFrameBackend) Resize(width, height int) bool {
	if b == nil || b.layer == 0 || width <= 0 || height <= 0 {
		return false
	}
	b.width, b.height = width, height
	metalResizeLayer(b.window, b.layer, width, height)
	return true
}

func (b *metalFrameBackend) imageBuffer(image *Image) int {
	if image == nil || len(image.Pixels) == 0 {
		return b.dummyBuffer
	}
	for i := 0; i < len(b.images); i++ {
		entry := &b.images[i]
		if entry.image != image {
			continue
		}
		if entry.revision == image.revision && entry.buffer != 0 {
			return entry.buffer
		}
		metalRelease(entry.buffer)
		entry.buffer = objcMsgBytes4(b.device, selector("newBufferWithBytes:length:options:"), image.Pixels, len(image.Pixels), metalResourceStorageShared)
		entry.revision = image.revision
		b.stats.TextureBytes += len(image.Pixels)
		return entry.buffer
	}
	buffer := objcMsgBytes4(b.device, selector("newBufferWithBytes:length:options:"), image.Pixels, len(image.Pixels), metalResourceStorageShared)
	if buffer != 0 {
		b.images = append(b.images, metalImageEntry{image: image, buffer: buffer, revision: image.revision})
		b.stats.TextureBytes += len(image.Pixels)
	}
	return buffer
}

func (b *metalFrameBackend) Render(list *drawList) bool {
	if b == nil || list == nil || b.width <= 0 || b.height <= 0 {
		return false
	}
	b.renderStarted = darwinNow()
	b.drawable = objcMsg0(b.layer, selector("nextDrawable"))
	if b.drawable == 0 {
		return false
	}
	texture := objcMsg0(b.drawable, selector("texture"))
	if texture == 0 {
		return false
	}
	vertexBytes := make([]byte, 0, len(list.vertices)*16)
	for i := 0; i < len(list.vertices); i++ {
		vertex := list.vertices[i]
		vertexBytes = metalAppendInt32(vertexBytes, int(vertex.x*256))
		vertexBytes = metalAppendInt32(vertexBytes, int(vertex.y*256))
		vertexBytes = metalAppendInt32(vertexBytes, int(vertex.u*256))
		vertexBytes = metalAppendInt32(vertexBytes, int(vertex.v*256))
	}
	frameBytes := make([]byte, 0, 8)
	frameBytes = metalAppendInt32(frameBytes, b.width)
	frameBytes = metalAppendInt32(frameBytes, b.height)
	b.vertexBuffer = objcMsgBytes4(b.device, selector("newBufferWithBytes:length:options:"), vertexBytes, len(vertexBytes), metalResourceStorageShared)
	b.frameBuffer = objcMsgBytes4(b.device, selector("newBufferWithBytes:length:options:"), frameBytes, len(frameBytes), metalResourceStorageShared)
	if b.vertexBuffer == 0 || b.frameBuffer == 0 {
		return false
	}
	b.command = objcMsg0(b.queue, selector("commandBuffer"))
	pass := objcMsg0(objcGetClass("MTLRenderPassDescriptor"), selector("renderPassDescriptor"))
	attachments := objcMsg0(pass, selector("colorAttachments"))
	attachment := objcMsg1(attachments, selector("objectAtIndexedSubscript:"), 0)
	objcMsg1(attachment, selector("setTexture:"), texture)
	objcMsg1(attachment, selector("setLoadAction:"), metalLoadActionDontCare)
	objcMsg1(attachment, selector("setStoreAction:"), metalStoreActionStore)
	encoder := objcMsg1(b.command, selector("renderCommandEncoderWithDescriptor:"), pass)
	if encoder == 0 {
		return false
	}
	objcMsg3(encoder, selector("setVertexBuffer:offset:atIndex:"), b.vertexBuffer, 0, 0)
	objcMsg3(encoder, selector("setVertexBuffer:offset:atIndex:"), b.frameBuffer, 0, 1)
	b.stats.TextureBytes = 0
	for commandIndex := 0; commandIndex < len(list.commands); commandIndex++ {
		command := list.commands[commandIndex]
		pipeline := b.blendPipeline
		if command.blend == BlendCopy {
			pipeline = b.copyPipeline
		}
		objcMsg1(encoder, selector("setRenderPipelineState:"), pipeline)
		var image *Image
		if command.image >= 0 && command.image < len(list.images) {
			image = list.images[command.image]
		}
		info := make([]byte, 0, metalCommandBytes)
		info = metalAppendInt32(info, int(command.color.R))
		info = metalAppendInt32(info, int(command.color.G))
		info = metalAppendInt32(info, int(command.color.B))
		info = metalAppendInt32(info, int(command.color.A))
		info = metalAppendInt32(info, int(command.kind))
		if image != nil {
			info = metalAppendInt32(info, image.Width)
			info = metalAppendInt32(info, image.Height)
			info = metalAppendInt32(info, image.Stride)
		} else {
			info = metalAppendInt32(info, 1)
			info = metalAppendInt32(info, 1)
			info = metalAppendInt32(info, 1)
		}
		info = metalAppendInt32(info, command.clip.minX)
		info = metalAppendInt32(info, command.clip.minY)
		info = metalAppendInt32(info, command.clip.maxX)
		info = metalAppendInt32(info, command.clip.maxY)
		info = metalAppendInt32(info, int(command.sampling))
		info = metalAppendInt32(info, 0)
		info = metalAppendInt32(info, 0)
		info = metalAppendInt32(info, 0)
		uniform := objcMsgBytes4(b.device, selector("newBufferWithBytes:length:options:"), info, len(info), metalResourceStorageShared)
		if uniform == 0 {
			return false
		}
		b.uniformBuffers = append(b.uniformBuffers, uniform)
		objcMsg3(encoder, selector("setFragmentBuffer:offset:atIndex:"), uniform, 0, 0)
		pixelBuffer := b.dummyBuffer
		if image != nil {
			pixelBuffer = b.imageBuffer(image)
		}
		objcMsg3(encoder, selector("setFragmentBuffer:offset:atIndex:"), pixelBuffer, 0, 1)
		objcMsg3(encoder, selector("drawPrimitives:vertexStart:vertexCount:"), metalPrimitiveTriangle, command.first, command.count)
	}
	objcMsg0(encoder, selector("endEncoding"))
	b.stats.SubmitSeconds = darwinNow() - b.renderStarted
	return true
}

func (b *metalFrameBackend) Present() bool {
	if b == nil || b.command == 0 || b.drawable == 0 {
		return false
	}
	started := darwinNow()
	texture := objcMsg0(b.drawable, selector("texture"))
	objcMsg0(texture, selector("retain"))
	metalRelease(b.lastTexture)
	b.lastTexture = texture
	objcMsg1(b.command, selector("presentDrawable:"), b.drawable)
	objcMsg0(b.command, selector("commit"))
	objcMsg0(b.command, selector("waitUntilCompleted"))
	b.stats.PresentSeconds = darwinNow() - started
	for i := 0; i < len(b.uniformBuffers); i++ {
		metalRelease(b.uniformBuffers[i])
	}
	b.uniformBuffers = b.uniformBuffers[:0]
	metalRelease(b.vertexBuffer)
	metalRelease(b.frameBuffer)
	b.vertexBuffer, b.frameBuffer = 0, 0
	b.command, b.drawable = 0, 0
	return true
}

func (b *metalFrameBackend) ReadPixels(surface *Surface) bool {
	if b == nil || b.lastTexture == 0 || surface == nil || b.width <= 0 || b.height <= 0 {
		return false
	}
	if surface.Width != b.width || surface.Height != b.height {
		surface.Resize(b.width, b.height)
	}
	bgra := make([]byte, surface.Stride*surface.Height)
	region := metalRegion{sizeX: b.width, sizeY: b.height, sizeZ: 1}
	metalGetBytes(b.lastTexture, selector("getBytes:bytesPerRow:fromRegion:mipmapLevel:"), bgra, surface.Stride, &region, 0)
	for y := 0; y < surface.Height; y++ {
		for x := 0; x < surface.Width; x++ {
			at := y*surface.Stride + x*4
			surface.Pixels[at] = bgra[at+2]
			surface.Pixels[at+1] = bgra[at+1]
			surface.Pixels[at+2] = bgra[at]
			surface.Pixels[at+3] = bgra[at+3]
		}
	}
	return true
}

func (b *metalFrameBackend) Destroy() {
	if b == nil {
		return
	}
	for i := 0; i < len(b.images); i++ {
		metalRelease(b.images[i].buffer)
	}
	b.images = nil
	for i := 0; i < len(b.uniformBuffers); i++ {
		metalRelease(b.uniformBuffers[i])
	}
	b.uniformBuffers = nil
	metalRelease(b.lastTexture)
	metalRelease(b.dummyBuffer)
	metalRelease(b.copyPipeline)
	metalRelease(b.blendPipeline)
	metalRelease(b.library)
	metalRelease(b.queue)
	metalRelease(b.layer)
	metalRelease(b.device)
	b.lastTexture, b.dummyBuffer = 0, 0
	b.copyPipeline, b.blendPipeline = 0, 0
	b.library, b.queue, b.layer = 0, 0, 0
	b.device = 0
}

func (b *metalFrameBackend) Stats() RenderStats {
	if b == nil {
		return RenderStats{}
	}
	return b.stats
}
