// Package imageproc resizes and re-encodes uploaded profile pictures so
// storage stays small without looking soft: input up to 4K is accepted,
// downscaled to a fixed max dimension with Lanczos resampling, and
// re-encoded at a quality level tuned for sharpness per byte.
package imageproc

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/disintegration/imaging"
)

const (
	maxInputDimension  = 4096 // reject anything above 4K on either axis
	maxOutputDimension = 512  // avatars don't need to be larger than this
	jpegQuality        = 88
)

// ProcessAvatar decodes, validates, resizes, and re-encodes an uploaded
// image. PNGs with an alpha channel stay PNG (to keep transparency);
// everything else becomes JPEG, which compresses far smaller than PNG for
// photographic content.
func ProcessAvatar(r io.Reader) (data []byte, contentType string, err error) {
	img, format, err := image.Decode(r)
	if err != nil {
		return nil, "", fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() > maxInputDimension || bounds.Dy() > maxInputDimension {
		return nil, "", fmt.Errorf("image exceeds max dimension of %dpx (got %dx%d)", maxInputDimension, bounds.Dx(), bounds.Dy())
	}

	resized := img
	if bounds.Dx() > maxOutputDimension || bounds.Dy() > maxOutputDimension {
		resized = imaging.Fit(img, maxOutputDimension, maxOutputDimension, imaging.Lanczos)
	}

	var buf bytes.Buffer
	if format == "png" && hasAlpha(resized) {
		if err := png.Encode(&buf, resized); err != nil {
			return nil, "", fmt.Errorf("encode png: %w", err)
		}
		return buf.Bytes(), "image/png", nil
	}

	if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, "", fmt.Errorf("encode jpeg: %w", err)
	}
	return buf.Bytes(), "image/jpeg", nil
}

func hasAlpha(img image.Image) bool {
	switch img.(type) {
	case *image.NRGBA, *image.RGBA, *image.NRGBA64, *image.RGBA64:
		bounds := img.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				_, _, _, a := img.At(x, y).RGBA()
				if a < 0xffff {
					return true
				}
			}
		}
	}
	return false
}
