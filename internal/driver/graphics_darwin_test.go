//go:build !renvo

package driver

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDarwinGraphicsHeadlessPixels(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("Darwin graphics integration test requires a Darwin/arm64 host")
	}
	repoRoot := driverRepoRoot(t)
	workDir := t.TempDir()
	appDir := filepath.Join(workDir, "cmd", "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "go.mod"), []byte("module example.com/graphicscheck\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := `package main

import (
	"graphics"
	"os"
)

func alphaAt(surface *graphics.Surface, x, y int) byte {
	return surface.Pixels[y*surface.Stride+x*4+3]
}

func main() {
	surface := graphics.NewSurface(64, 64)
	surface.SetTranslation(4, 4)
	surface.FillRect(graphics.R(0, 0, 8, 8), graphics.RGBA(255, 255, 255, 255))
	if alphaAt(surface, 6, 5) != 255 {
		print("FAIL translation six five\n")
		return
	}
	if alphaAt(surface, 5, 5) != 255 {
		print("FAIL translation five five\n")
		return
	}
	if alphaAt(surface, 1, 1) != 0 {
		print("FAIL translation origin\n")
		return
	}
	identity := graphics.Identity()
	surface.SetTransform(&identity)
	surface.Clear(graphics.RGBA(0, 0, 0, 0))
	var path graphics.Path
	path.MoveTo(graphics.Point{X: 8, Y: 48})
	path.QuadTo(graphics.Point{X: 32, Y: 8}, graphics.Point{X: 56, Y: 48})
	path.LineTo(graphics.Point{X: 48, Y: 56})
	path.QuadTo(graphics.Point{X: 32, Y: 32}, graphics.Point{X: 16, Y: 56})
	path.Close()
	if !path.Contains(graphics.Point{X: 32.5, Y: 40.5}, graphics.FillEvenOdd) {
		print("FAIL contains\n")
		return
	}
	surface.FillPath(&path, graphics.FillEvenOdd, graphics.RGBA(255, 255, 255, 255))
	if alphaAt(surface, 32, 40) != 255 || alphaAt(surface, 4, 4) != 0 {
		print("FAIL path fill\n")
		return
	}
	surface.Clear(graphics.RGBA(0, 0, 0, 0))
	surface.ResetTransform()
	surface.SetAxisX(2, 0)
	surface.SetAxisY(0, 2)
	surface.SetOffset(1, 1)
	surface.FillRect(graphics.R(0, 0, 2, 2), graphics.RGBA(255, 255, 255, 255))
	if alphaAt(surface, 2, 2) != 255 {
		if alphaAt(surface, 1, 1) == 255 {
			print("FAIL affine inside at 1\n")
			return
		}
		if alphaAt(surface, 4, 4) == 255 {
			print("FAIL affine inside at 4\n")
			return
		}
		if alphaAt(surface, 8, 8) == 255 {
			print("FAIL affine inside at 8\n")
			return
		}
		print("FAIL affine inside\n")
		return
	}
	if alphaAt(surface, 0, 0) != 0 {
		print("FAIL affine origin\n")
		return
	}
	if alphaAt(surface, 5, 5) != 0 {
		if alphaAt(surface, 10, 10) == 255 {
			print("FAIL affine outside through 10\n")
			return
		}
		print("FAIL affine outside\n")
		return
	}
	surface.Clear(graphics.RGBA(0, 0, 0, 0))
	surface.SetAffine(2, 0, 0, 2, 1, 1)
	affinePoint := surface.TransformPoint(graphics.Point{X: 2, Y: 2})
	if int(affinePoint.X) != 5 || int(affinePoint.Y) != 5 {
		print("FAIL set affine point\n")
		return
	}
	surface.FillRect(graphics.R(0, 0, 2, 2), graphics.RGBA(255, 255, 255, 255))
	if alphaAt(surface, 2, 2) != 255 || alphaAt(surface, 0, 0) != 0 || alphaAt(surface, 5, 5) != 0 {
		print("FAIL set affine draw\n")
		return
	}
	surface.Clear(graphics.RGBA(0, 0, 0, 0))
	matrix := graphics.Mat2x3{A: 2, D: 2, TX: 1, TY: 1}
	surface.SetTransform(&matrix)
	matrixPoint := surface.TransformPoint(graphics.Point{X: 2, Y: 2})
	if int(matrixPoint.X) != 5 || int(matrixPoint.Y) != 5 {
		if int(matrixPoint.X) == 17 {
			print("FAIL matrix transform point 17\n")
			return
		}
		if int(matrixPoint.X) == 8 {
			print("FAIL matrix transform point 8\n")
			return
		}
		print("FAIL matrix transform point\n")
		return
	}
	surface.FillRect(graphics.R(0, 0, 2, 2), graphics.RGBA(255, 255, 255, 255))
	if alphaAt(surface, 2, 2) != 255 || alphaAt(surface, 0, 0) != 0 || alphaAt(surface, 5, 5) != 0 {
		print("FAIL matrix transform draw\n")
		return
	}
	surface.Clear(graphics.RGBA(0, 0, 0, 0))
	image := graphics.NewImage(1, 1, []byte{255, 255, 255, 255})
	surface.DrawImage(image, graphics.R(0, 0, 1, 1), graphics.R(0, 0, 1, 1), graphics.SamplingNearest, graphics.White)
	if alphaAt(surface, 1, 1) != 255 || alphaAt(surface, 2, 2) != 255 || alphaAt(surface, 0, 0) != 0 || alphaAt(surface, 3, 3) != 0 {
		print("FAIL transformed image\n")
		return
	}
	surface.Clear(graphics.RGBA(0, 0, 0, 0))
	surface.SetTranslation(4, 5)
	var rectangle graphics.Path
	rectangle.MoveTo(graphics.Point{X: 0, Y: 0})
	rectangle.LineTo(graphics.Point{X: 4, Y: 0})
	rectangle.LineTo(graphics.Point{X: 4, Y: 4})
	rectangle.LineTo(graphics.Point{X: 0, Y: 4})
	rectangle.Close()
	surface.FillPath(&rectangle, graphics.FillNonZero, graphics.RGBA(255, 255, 255, 255))
	if alphaAt(surface, 5, 6) != 255 || alphaAt(surface, 1, 1) != 0 {
		print("FAIL translated path\n")
		return
	}
	surface.ResetTransform()
	surface.Clear(graphics.RGBA(0, 0, 0, 0))
	surface.DrawLine(graphics.Point{X: 0, Y: 0}, graphics.Point{X: 32, Y: 32}, 2, graphics.RGBA(255, 255, 255, 255))
	if alphaAt(surface, 5, 5) != 255 || alphaAt(surface, 18, 18) != 255 || alphaAt(surface, 27, 27) != 255 {
		print("FAIL line\n")
		return
	}
	surface.Clear(graphics.RGBA(0, 0, 0, 0))
	surface.SetTranslation(2, 3)
	surface.PushTransform()
	surface.ResetTransform()
	surface.PopTransform()
	surface.FillRect(graphics.R(0, 0, 2, 2), graphics.RGBA(255, 255, 255, 255))
	if alphaAt(surface, 2, 3) != 255 || alphaAt(surface, 0, 0) != 0 {
		print("FAIL transform stack\n")
		return
	}
	surface.ResetTransform()
	surface.Clear(graphics.RGBA(0, 0, 0, 0))
	mask := graphics.NewMask(1, 1, []byte{128})
	surface.DrawImage(mask, graphics.R(0, 0, 1, 1), graphics.R(2, 2, 2, 2), graphics.SamplingNearest, graphics.RGBA(255, 0, 0, 255))
	if alphaAt(surface, 2, 2) != 128 {
		print("FAIL mask\n")
		return
	}
	surface.Clear(graphics.RGBA(0, 0, 0, 0))
	tile := graphics.NewImage(2, 2, []byte{
		255, 0, 0, 255, 0, 255, 0, 255,
		0, 0, 255, 255, 255, 255, 255, 255,
	})
	surface.DrawImage(tile, graphics.R(0, 0, 2, 2), graphics.R(2, 2, 4, 4), graphics.SamplingNearest, graphics.White)
	topLeft := 2*surface.Stride + 2*4
	topRight := 2*surface.Stride + 5*4
	bottomLeft := 5*surface.Stride + 2*4
	bottomRight := 5*surface.Stride + 5*4
	if surface.Pixels[topLeft] != 255 || surface.Pixels[topLeft+1] != 0 || surface.Pixels[topLeft+2] != 0 {
		print("FAIL scaled image red\n")
		return
	}
	if surface.Pixels[topRight] != 0 || surface.Pixels[topRight+1] != 255 || surface.Pixels[topRight+2] != 0 {
		print("FAIL scaled image green\n")
		return
	}
	if surface.Pixels[bottomLeft] != 0 || surface.Pixels[bottomLeft+1] != 0 || surface.Pixels[bottomLeft+2] != 255 {
		print("FAIL scaled image blue\n")
		return
	}
	if surface.Pixels[bottomRight] != 255 || surface.Pixels[bottomRight+1] != 255 || surface.Pixels[bottomRight+2] != 255 {
		print("FAIL scaled image white\n")
		return
	}
	surface.FillEllipse(graphics.R(8, 8, 12, 8), graphics.RGBA(255, 255, 255, 255))
	if alphaAt(surface, 14, 12) != 255 {
		print("FAIL ellipse inside\n")
		return
	}
	if alphaAt(surface, 7, 7) != 0 {
		print("FAIL ellipse outside\n")
		return
	}
	font := graphics.NewBuiltinFont(1)
	metrics := graphics.MeasureText(font, "Hi")
	if int(metrics.Width) != 12 {
		print("FAIL measure\n")
		return
	}
	surface.DrawText(font, graphics.Point{X: 24, Y: 12}, "Hi", graphics.RGBA(255, 255, 255, 255))
	if alphaAt(surface, 24, 5) != 255 {
		print("FAIL text\n")
		return
	}
	surface.Clear(graphics.RGBA(0, 0, 0, 0))
	surface.ResetTransform()
	surface.DrawText(font, graphics.Point{X: 2, Y: 9}, "A", graphics.RGBA(255, 255, 255, 255))
	textPixels := 0
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			if alphaAt(surface, x, y) != 0 {
				textPixels++
				if x < 2 || x >= 7 || y < 2 || y >= 9 {
					print("FAIL text bounds\n")
					return
				}
			}
		}
	}
	if textPixels != 18 {
		print("FAIL text pixels\n")
		return
	}
	fontData, readError := os.ReadFile("/System/Library/Fonts/SFNSMono.ttf")
	if readError != nil {
		print("FAIL read TrueType\n")
		return
	}
	smoothFont, fontError := graphics.NewTrueTypeFont(fontData, 20)
	if fontError != nil {
		print("FAIL parse TrueType\n")
		return
	}
	smoothMetrics := graphics.MeasureText(smoothFont, "Smooth AV")
	if smoothMetrics.Width < 80 || smoothMetrics.Height != 20 {
		print("FAIL TrueType metrics ")
		print(int(smoothMetrics.Width))
		print(" ")
		print(int(smoothMetrics.Height))
		print("\n")
		return
	}
	surface.Clear(graphics.Transparent)
	surface.DrawText(smoothFont, graphics.Point{X: 2, Y: 24}, "Smooth", graphics.White)
	smoothPixels := 0
	partialPixels := 0
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			alpha := alphaAt(surface, x, y)
			if alpha != 0 {
				smoothPixels++
			}
			if alpha > 0 && alpha < 255 {
				partialPixels++
			}
		}
	}
	if smoothPixels == 0 || partialPixels == 0 {
		print("FAIL TrueType antialiasing\n")
		return
	}
	ppm := surface.EncodePPM()
	if len(ppm) != 12301 || ppm[0] != 'P' || ppm[1] != '6' || ppm[12] != '\n' {
		print("FAIL PPM export\n")
		return
	}
	surface.Clear(graphics.RGBA(0, 0, 0, 0))
	surface.SetAffine(0.75, 0.5, -0.5, 0.75, 32, 32)
	surface.FillRect(graphics.R(-4, -4, 8, 8), graphics.RGBA(255, 255, 255, 255))
	transformPixels := 0
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			if alphaAt(surface, x, y) != 0 {
				transformPixels++
			}
		}
	}
	if transformPixels < 30 || alphaAt(surface, 32, 32) != 255 {
		print("FAIL fractional affine\n")
		return
	}
	print("PASS\n")
}
`
	if err := os.WriteFile(filepath.Join(appDir, "main.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	backend := filepath.Join(t.TempDir(), "renvo-backend")
	build := exec.Command("go", "build", "-o", backend, "./backend")
	build.Dir = repoRoot
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("backend build failed: %v\n%s", err, output)
	}
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)
	result := RunCommand(
		[]string{"renvo", "-t", "darwin/arm64", "-o", "app", "./cmd/app"},
		[]string{BackendEnv + "=" + backend, StdRootEnv + "=" + filepath.Join(repoRoot, "std")},
		nil,
	)
	if !result.Ok {
		if len(result.Compile.Build.Unit) > 0 {
			cmd := exec.Command(backend, "-t", "darwin/arm64", "-o", "-", "-")
			cmd.Stdin = bytes.NewReader(result.Compile.Build.Unit)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Logf("backend diagnostic: %v\n%s", err, output)
			}
		}
		t.Fatalf("graphics compilation failed: err=%d path=%q buildErr=%d arg=%q errorPath=%q at=%d package=%d file=%d token=%d", result.Error, result.ErrorPath, result.Compile.Build.Error, result.Compile.Build.ErrorArg, result.Compile.Build.ErrorPath, result.Compile.Build.ErrorAt, result.Compile.Build.ErrorPackage, result.Compile.Build.ErrorFile, result.Compile.Build.ErrorToken)
	}
	output, err := exec.Command(filepath.Join(workDir, "app")).CombinedOutput()
	if err != nil {
		t.Fatalf("graphics program failed: %v\n%s", err, output)
	}
	if string(output) != "PASS\n" {
		t.Fatalf("graphics output = %q", output)
	}
}
