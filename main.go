package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("error while running the app: %v", err)
	}
}

const (
	ppmMaxVal = 0xFF
)

func run() error {
	const (
		outputWidth  = 1000
		outputHeigth = 1000
	)
	var pixels [outputWidth * outputHeigth]uint64

	const frameCount = 10
	const framePattern = "frames/frame_%d.ppm"
	for frame := range frameCount {
		err := renderRgbSquaresFrame(frame, outputHeigth, outputHeigth, pixels[:])
		if err != nil {
			return fmt.Errorf("failed to render frame %d: %w", frame, err)
		}

		frameOutputPath := fmt.Sprintf(framePattern, frame)
		err = writePPM(frameOutputPath, outputWidth, outputHeigth, pixels[:])
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", frameOutputPath, err)
		}
	}

	renderOutputPath := "output.mp4"
	if err := renderFrames(renderOutputPath, framePattern); err != nil {
		return fmt.Errorf("failed to render frames with pattern %s to : %w", framePattern, renderOutputPath, err)
	}

	return nil
}

func renderRgbSquaresFrame(frameN, width, height int, pixels []uint64) error {
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
			squareHeightEnd += squareWidth
			color = (color + 1) % colorCount
		}
		squareWidthEnd := squareWidth
		for x := range width {
			if x > squareWidthEnd {
				squareWidthEnd += squareWidth
				color = (color + 1) % colorCount
			}
			i := y*height + x
			switch color {
			case 0:
				pixels[i] = pixelToRgba(ppmMaxVal, 0, 0, ppmMaxVal)
			case 1:
				pixels[i] = pixelToRgba(0, ppmMaxVal, 0, ppmMaxVal)
			case 2:
				pixels[i] = pixelToRgba(0, 0, ppmMaxVal, ppmMaxVal)
			}
		}
	}

	return nil
}

func pixelToRgba(r, g, b, a uint8) uint64 {
	return uint64(r)<<(8*0) | uint64(g)<<(8*1) | uint64(b)<<(8*2) | uint64(a)<<(8*3)
}

func rgbaFromPixel(p uint64) (r, g, b, a uint8) {
	return uint8(p >> (8 * 0)), uint8(p >> (8 * 1)), uint8(p >> (8 * 2)), uint8(p >> (8 * 3))
}

func writePPM(path string, width, height uint, pixels []uint64) (err error) {
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
		r, g, b, _ := rgbaFromPixel(p)
		pixelBytes = append(pixelBytes, r, g, b)
	}

	_, err = f.Write(pixelBytes)
	if err != nil {
		return fmt.Errorf("failed to write pixels: %w", err)
	}

	return nil
}

func renderFrames(path string, framePattern string) error {
	return exec.Command("ffmpeg",
		"-y", // force file override
		"-framerate", "5",
		"-start_number", "0",
		"-i", framePattern,
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		path,
	).Run()
}
