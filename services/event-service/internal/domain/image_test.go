package domain

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func TestSanitizeImageStripsTrailingMetadata(t *testing.T) {
	photo := image.NewRGBA(image.Rect(0, 0, 20, 20))
	photo.Set(1, 1, color.RGBA{R: 255, A: 255})
	var raw bytes.Buffer
	if err := jpeg.Encode(&raw, photo, nil); err != nil {
		t.Fatal(err)
	}
	input := append(raw.Bytes(), []byte("PRIVATE_METADATA_MARKER")...)
	safe, err := SanitizeImage(input)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(safe, []byte("PRIVATE_METADATA_MARKER")) {
		t.Fatal("source metadata leaked")
	}
	if _, err = jpeg.Decode(bytes.NewReader(safe)); err != nil {
		t.Fatalf("sanitized image invalid: %v", err)
	}
}

func TestSanitizeImageRejectsOversizeAndInvalid(t *testing.T) {
	if _, err := SanitizeImage(make([]byte, MaxImageInputBytes+1)); err == nil {
		t.Fatal("oversize image accepted")
	}
	if _, err := SanitizeImage([]byte("not a jpeg")); err == nil {
		t.Fatal("invalid image accepted")
	}
}
