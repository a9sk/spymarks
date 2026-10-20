package image

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// Load decodes an image file to RGBA.
func Load(path string) (*image.RGBA, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("opening image: %w", err)
	}
	defer f.Close()

	img, format, err := image.Decode(f)
	if err != nil {
		return nil, "", fmt.Errorf("decoding image: %w", err)
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}
	return rgba, format, nil
}

// Save writes an RGBA image to path, choosing the encoder by extension.
func Save(path string, img *image.RGBA, format string) error {
	return SaveWithQuality(path, img, format, 95)
}

// SaveWithQuality writes an RGBA image to path with the requested JPEG quality.
// Quality is ignored for PNG output.
func SaveWithQuality(path string, img *image.RGBA, format string, quality int) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating output image: %w", err)
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".jpg" || ext == ".jpeg" {
		if quality <= 0 {
			quality = 95
		}
		if quality > 100 {
			quality = 100
		}
		return EncodeJPEG(f, img, quality)
	}
	return png.Encode(f, img)
}

// EncodeJPEG writes img to w as JPEG with the given quality.
func EncodeJPEG(w *os.File, img *image.RGBA, quality int) error {
	if quality <= 0 {
		quality = 95
	}
	if quality > 100 {
		quality = 100
	}
	return jpeg.Encode(w, img, &jpeg.Options{Quality: quality})
}

// ToGray converts an RGBA image to a grayscale float64 slice in [0,255].
func ToGray(img *image.RGBA) []float64 {
	return ToChannel(img, 'y')
}

// FromGray replaces the luminance of img with the grayscale values in gray.
func FromGray(img *image.RGBA, gray []float64) {
	FromChannel(img, 'y', gray)
}

// ToChannel extracts a single channel as float64 values in [0,255].
// Supported channels: 'r', 'g', 'b', 'a', 'y' (luminance),
// 'c' (Cb) and 'C' (Cr).
func ToChannel(img *image.RGBA, ch byte) []float64 {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	out := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			var v float64
			switch ch {
			case 'r':
				v = float64(c.R)
			case 'g':
				v = float64(c.G)
			case 'b':
				v = float64(c.B)
			case 'a':
				v = float64(c.A)
			case 'y':
				v = math.Floor(0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B) + 0.5)
			case 'c':
				v = math.Floor(128.0 - 0.168736*float64(c.R) - 0.331264*float64(c.G) + 0.5*float64(c.B) + 0.5)
			case 'C':
				v = math.Floor(128.0 + 0.5*float64(c.R) - 0.419688*float64(c.G) - 0.081312*float64(c.B) + 0.5)
			default:
				v = float64(c.G)
			}
			out[y*w+x] = v
		}
	}
	return out
}

// FromChannel writes values back into a single channel of img.
// Supported channels: 'r', 'g', 'b', 'a', 'y' (luminance),
// 'c' (Cb) and 'C' (Cr).
func FromChannel(img *image.RGBA, ch byte, values []float64) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			v := values[y*w+x]
			switch ch {
			case 'r':
				c.R = clampf(v)
			case 'g':
				c.G = clampf(v)
			case 'b':
				c.B = clampf(v)
			case 'a':
				c.A = clampf(v)
			case 'y':
				oldLum := 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
				delta := v - oldLum
				c.R = clampf(float64(c.R) + delta)
				c.G = clampf(float64(c.G) + delta)
				c.B = clampf(float64(c.B) + delta)
			case 'c', 'C':
				y := 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
				cb := 128.0 - 0.168736*float64(c.R) - 0.331264*float64(c.G) + 0.5*float64(c.B)
				cr := 128.0 + 0.5*float64(c.R) - 0.419688*float64(c.G) - 0.081312*float64(c.B)
				if ch == 'c' {
					cb = v
				} else {
					cr = v
				}
				c.R = clampf(y + 1.402*(cr-128.0))
				c.G = clampf(y - 0.344136*(cb-128.0) - 0.714136*(cr-128.0))
				c.B = clampf(y + 1.772*(cb-128.0))
			default:
				c.G = clampf(v)
			}
			img.SetRGBA(bounds.Min.X+x, bounds.Min.Y+y, c)
		}
	}
}

func clampf(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}

// DetectFormatFromExt returns image format from extension.
func DetectFormatFromExt(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".jpg" || ext == ".jpeg" {
		return "jpeg"
	}
	return "png"
}
