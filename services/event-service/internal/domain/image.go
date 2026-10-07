package domain

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
)

const MaxImageInputBytes = 650 * 1024
const MaxImageStoredBytes = 1024 * 1024

var ErrInvalidImage = errors.New("image must be a valid JPEG no larger than 650 KiB and 1600 by 1600 pixels")

// SanitizeImage re-encodes accepted JPEGs, removing EXIF and other embedded metadata.
func SanitizeImage(data []byte) ([]byte, error) {
	if len(data) == 0 || len(data) > MaxImageInputBytes {
		return nil, ErrInvalidImage
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 1600 || cfg.Height > 1600 {
		return nil, ErrInvalidImage
	}
	decoded, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrInvalidImage
	}
	canvas := image.NewRGBA(decoded.Bounds())
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(canvas, canvas.Bounds(), decoded, decoded.Bounds().Min, draw.Over)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, canvas, &jpeg.Options{Quality: 80}); err != nil || out.Len() > MaxImageStoredBytes {
		return nil, ErrInvalidImage
	}
	return out.Bytes(), nil
}
