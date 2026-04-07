package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/gif"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pechorka/simple-raytracer/pkg/octree"
	"github.com/pechorka/simple-raytracer/pkg/utils"
)

type frameRenderer func(frameN, frameCount, width, height int, pixels []uint32) error

type videoExporter func(
	path string,
	width, height int,
	frameCount, fps int,
	frameRenderer frameRenderer,
) error

func main() {
	if err := run(); err != nil {
		log.Fatalf("error while running the app: %v", err)
	}
}

const (
	ppmMaxVal = 0xFF
)

func run() error {
	frameCount := flag.Int("fc", 100, "number of frames to generate")
	fps := flag.Int("fps", 30, "target video fps")
	rendererName := flag.String("renderer", "mandelbrot", "renderer to use: rgb-squares, plasma, mandelbrot, tunnel")
	videoExporterName := flag.String("video-exporter", "ffmpeg", "video-exporter to use: ffmpeg, gif")
	flag.Parse()

	renderers := map[string]frameRenderer{
		"rgb-squares": renderRgbSquaresFrame,
		"plasma":      renderPlasmaFrame,
		"mandelbrot":  renderMandelbrotFrame,
		"tunnel":      renderTunnelFrame,
	}
	renderer, ok := renderers[strings.ToLower(*rendererName)]
	if !ok {
		return fmt.Errorf("unsupported renderer %q", *rendererName)
	}

	videoExporters := map[string]struct {
		exporter videoExporter
		ext      string
	}{
		"ffmpeg": {
			exporter: renderFfmpeg,
			ext:      "mp4",
		},
		"gif": {
			exporter: renderGIF,
			ext:      "gif",
		},
	}
	videoExporterMeta, ok := videoExporters[strings.ToLower(*videoExporterName)]
	if !ok {
		return fmt.Errorf("unsupported video-exporter %q", *videoExporterName)
	}

	const (
		outputWidth  = 1000
		outputHeigth = 1000
	)

	renderOutputPath := "output." + videoExporterMeta.ext
	err := videoExporterMeta.exporter(renderOutputPath, outputWidth, outputHeigth, *frameCount, *fps, renderer)
	if err != nil {
		return fmt.Errorf("failed to render frames as gif to %s: %w", renderOutputPath, err)
	}

	return nil
}

func renderRgbSquaresFrame(frameN, frameCount, width, height int, pixels []uint32) error {
	if len(pixels) != height*width {
		return fmt.Errorf("pixels buffer has len %d, but expect to have %d", len(pixels), height*width)
	}

	squareWidth := width / 10
	squareHeight := height / 10

	squareHeightEnd := squareHeight
	const colorCount = 3
	color := frameN % colorCount
	for y := range height {
		if y > squareHeightEnd {
			squareHeightEnd += squareHeight
			color = (color + 1) % colorCount
		}
		squareWidthEnd := squareWidth
		for x := range width {
			if x > squareWidthEnd {
				squareWidthEnd += squareWidth
				color = (color + 1) % colorCount
			}
			i := y*width + x
			switch color {
			case 0:
				pixels[i] = utils.PixelToRGBA(ppmMaxVal, 0, 0, ppmMaxVal)
			case 1:
				pixels[i] = utils.PixelToRGBA(0, ppmMaxVal, 0, ppmMaxVal)
			case 2:
				pixels[i] = utils.PixelToRGBA(0, 0, ppmMaxVal, ppmMaxVal)
			}
		}
	}

	return nil
}

func renderPlasmaFrame(frameN, frameCount, width, height int, pixels []uint32) error {
	if len(pixels) != height*width {
		return fmt.Errorf("pixels buffer has len %d, but expect to have %d", len(pixels), height*width)
	}
	t := float64(frameN) / float64(frameCount)
	for y := range height {
		for x := range width {
			xf := (float64(x)/float64(width))*2 - 1 // normalize to -1..1
			yf := (float64(y)/float64(height))*2 - 1

			v1 := math.Sin(xf*10 + t*2*math.Pi)
			v2 := math.Sin(yf*10 + t*2*math.Pi)
			v3 := math.Sin((xf+yf)*10 + t*2*math.Pi)
			v4 := math.Sin(math.Sqrt(xf*xf+yf*yf)*10 - t*2*math.Pi)

			v := (v1 + v2 + v3 + v4) / 4
			r := (math.Sin(v*math.Pi) + 1) / 2 * 255 // normalize to 0..255
			g := (math.Sin(v*math.Pi+2*math.Pi/3) + 1) / 2 * 255
			b := (math.Sin(v*math.Pi+4*math.Pi/3) + 1) / 2 * 255

			i := y*width + x
			pixels[i] = utils.PixelToRGBA(uint8(r), uint8(g), uint8(b), ppmMaxVal)
		}
	}

	return nil
}

func renderMandelbrotFrame(frameN, frameCount, width, height int, pixels []uint32) error {
	if len(pixels) != height*width {
		return fmt.Errorf("pixels buffer has len %d, but expect to have %d", len(pixels), height*width)
	}

	// Zoom into this interesting point over time
	centerR, centerI := -0.745, 0.186
	zoom := math.Pow(0.9, float64(frameN)) // each frame zooms in ~10%
	maxIter := 100

	for y := range height {
		for x := range width {
			// Map pixel to complex plane, centered on our target
			cr := centerR + (float64(x)/float64(width)*2-1)*zoom*1.5
			ci := centerI + (float64(y)/float64(height)*2-1)*zoom*1.5

			zr, zi := 0.0, 0.0
			iter := 0
			for iter < maxIter {
				newZr := zr*zr - zi*zi + cr
				newZi := 2*zr*zi + ci
				zr, zi = newZr, newZi
				if zr*zr+zi*zi > 4 {
					break
				}
				iter++
			}

			t := float64(iter) / float64(maxIter)
			r := (math.Sin(t*math.Pi*3) + 1) / 2 * 255
			g := (math.Sin(t*math.Pi*3+2*math.Pi/3) + 1) / 2 * 255
			b := (math.Sin(t*math.Pi*3+4*math.Pi/3) + 1) / 2 * 255
			if iter == maxIter {
				r, g, b = 0, 0, 0 // points inside the set are black
			}

			i := y*width + x
			pixels[i] = utils.PixelToRGBA(uint8(r), uint8(g), uint8(b), ppmMaxVal)
		}
	}
	return nil
}

func renderTunnelFrame(frameN, frameCount, width, height int, pixels []uint32) error {
	if len(pixels) != height*width {
		return fmt.Errorf("pixels buffer has len %d, but expect to have %d", len(pixels), height*width)
	}

	t := float64(frameN) / float64(frameCount) * 2 * math.Pi

	for y := range height {
		for x := range width {
			// Distance and angle from center
			dx := float64(x) - float64(width)/2
			dy := float64(y) - float64(height)/2
			dist := math.Sqrt(dx*dx + dy*dy)
			angle := math.Atan2(dy, dx)

			if dist < 1 {
				dist = 1 // avoid division by zero
			}

			// Texture coordinates: depth + rotation
			u := 1.0/dist*float64(width)*0.1 + t
			v := angle/math.Pi + t*0.3

			// Checkerboard-ish pattern in tunnel space
			v1 := math.Sin(u*6) * math.Cos(v*6)
			v2 := math.Sin(u*3+v*5) * 0.5

			val := (v1 + v2 + 1) / 3

			// Fade to black at edges and center for depth feel
			fade := math.Exp(-dist / float64(width) * 3)

			r := (math.Sin(val*math.Pi*2) + 1) / 2 * 255 * fade
			g := (math.Sin(val*math.Pi*2+2) + 1) / 2 * 255 * fade
			b := (math.Sin(val*math.Pi*2+4) + 1) / 2 * 255 * fade

			i := y*width + x
			pixels[i] = utils.PixelToRGBA(uint8(r), uint8(g), uint8(b), ppmMaxVal)
		}
	}
	return nil
}

func writePPM(path string, width, height int, pixels []uint32) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to open file %q: %w", path, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close file %s: %w", path, cerr))
		}
	}()

	_, err = fmt.Fprintf(f, `P6
%d %d
%d
`, width, height, ppmMaxVal)
	if err != nil {
		return fmt.Errorf("failed to write ppm header: %w", err)
	}

	pixelBytes := make([]byte, 0, len(pixels)*3)
	for _, p := range pixels {
		r, g, b, _ := utils.RGBAFromPixel(p)
		pixelBytes = append(pixelBytes, r, g, b)
	}

	_, err = f.Write(pixelBytes)
	if err != nil {
		return fmt.Errorf("failed to write pixels: %w", err)
	}

	return nil
}

// TODO: Replace ffmpeg rendering with Y4M (YUV4MPEG2) output
// Header is:
//
//	"YUV4MPEG2 W%d H%d F%d:1 Ip A1:1 C444\n"
//	(W=width, H=height, F=fps:1, Ip=progressive, A=square pixels, C444=no chroma subsampling)
//
// For each frame:
//  1. Write "FRAME\n" (literal 6 bytes)
//  2. Write Y plane: for each pixel in row order, convert RGB to Y and write one byte
//     Y = clamp(0.299*R + 0.587*G + 0.114*B, 0, 255)
//  3. Write U plane: same order, one byte per pixel
//     U = clamp(-0.169*R - 0.331*G + 0.500*B + 128, 0, 255)
//  4. Write V plane: same order, one byte per pixel
//     V = clamp(0.500*R - 0.419*G - 0.081*B + 128, 0, 255)
//     Each plane is width*height bytes. Total per frame: width*height*3 bytes.
func renderFfmpeg(
	path string,
	width, height int,
	frameCount, fps int,
	frameRenderer frameRenderer,
) error {
	pixels := make([]uint32, width*height)

	const framesFolder = "frames"
	err := os.MkdirAll(framesFolder, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create folder for frames %q: %w", framesFolder, err)
		}

	framePattern := filepath.Join(framesFolder, "/frame_%d.ppm")
	for frame := range frameCount {
		err := frameRenderer(frame, frameCount, width, height, pixels)
		if err != nil {
			return fmt.Errorf("failed to render frame %d: %w", frame, err)
		}

		frameOutputPath := fmt.Sprintf(framePattern, frame)
		err = writePPM(frameOutputPath, width, height, pixels[:])
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", frameOutputPath, err)
		}
	}

	err := exec.Command("ffmpeg",
		"-y", // force file override
		"-framerate", strconv.Itoa(fps),
		"-start_number", "0",
		"-i", framePattern,
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		path,
	).Run()
	if err != nil {
		return fmt.Errorf(
			"failed to render ppms with ffmpeg in to %q (ppms are left available for inspection in %q): %w",
			path, framesFolder, err,
		)
	}

	if err := os.RemoveAll(framesFolder); err != nil {
		return fmt.Errorf("failed to cleanup frames from folder %s: %w", framesFolder, err)
	}

	return nil
}

func renderGIF(
	path string,
	width, height int,
	frameCount, fps int,
	frameRenderer frameRenderer,
) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to open file %q: %w", path, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close file %s: %w", path, cerr))
		}
	}()

	pixels := make([]uint32, width*height)

	res := gif.GIF{}
	frameRect := image.Rectangle{
		Max: image.Point{
			X: width,
			Y: height,
		},
	}
	frameDelay := int(100 / fps)
	for frameN := range frameCount {
		log.Printf("INFO rendering frame %d\n", frameN)
		err := frameRenderer(frameN, frameCount, width, height, pixels)
		if err != nil {
			return fmt.Errorf("failed to render frame %d: %w", frameN, err)
		}

		root := octree.NewRoot()

		for i, p := range pixels {
			if err := root.Insert(p, i); err != nil {
				return fmt.Errorf("failed to insert pixel %d into octree: %w", i, err)
			}
		}

		frame, err := root.ToImage(frameRect)
		if err != nil {
			return fmt.Errorf("failed to convert octree to frame %d: %w", frameN, err)
		}
		res.Image = append(res.Image, frame)
		res.Delay = append(res.Delay, frameDelay)
	}

	err = gif.EncodeAll(f, &res)
	if err != nil {
		return fmt.Errorf("failed to encode gif: %w", err)
	}

	return nil
}
