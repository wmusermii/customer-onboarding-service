package service

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"

	"customer-onboarding-service/internal/domain"
)

const (
	maxImageBytes = 5 << 20
	maxVideoBytes = 8 << 20 // asumsi: dokumen tidak menyebut batas video
	minShortSide  = 600     // resolusi minimum 800x600, orientasi bebas
	minLongSide   = 800
	maxSide       = 8000 // cegah decompression bomb
)

func badFile(status int, name, msg string) error {
	return domain.ErrInvalidFile.WithHTTP(status).WithDetail(name + ": " + msg)
}

// sanitizeImage: validasi format/ukuran/resolusi, lalu re-encode
// supaya EXIF (termasuk GPS) terbuang.
func sanitizeImage(name string, data []byte) ([]byte, string, error) {
	if len(data) == 0 {
		return nil, "", badFile(http.StatusBadRequest, name, "empty file")
	}
	if len(data) > maxImageBytes {
		return nil, "", badFile(http.StatusRequestEntityTooLarge, name, "max 5 MB")
	}

	var ext string
	switch http.DetectContentType(data) {
	case "image/jpeg":
		ext = "jpg"
	case "image/png":
		ext = "png"
	default:
		return nil, "", badFile(http.StatusUnsupportedMediaType, name, "only JPEG or PNG allowed")
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", badFile(http.StatusUnprocessableEntity, name, "unreadable image")
	}
	short, long := cfg.Width, cfg.Height
	if short > long {
		short, long = long, short
	}
	if short < minShortSide || long < minLongSide {
		return nil, "", badFile(http.StatusUnprocessableEntity, name, "minimum resolution is 800x600")
	}
	if long > maxSide {
		return nil, "", badFile(http.StatusUnprocessableEntity, name, "resolution too large")
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", badFile(http.StatusUnprocessableEntity, name, "unreadable image")
	}
	var buf bytes.Buffer
	if ext == "jpg" {
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 92})
	} else {
		err = png.Encode(&buf, img)
	}
	if err != nil {
		return nil, "", err
	}
	return buf.Bytes(), ext, nil
}

func validateVideo(name string, data []byte) error {
	if len(data) > maxVideoBytes {
		return badFile(http.StatusRequestEntityTooLarge, name, "max 8 MB")
	}
	if len(data) < 12 || string(data[4:8]) != "ftyp" { // signature MP4
		return badFile(http.StatusUnsupportedMediaType, name, "only MP4 allowed")
	}
	return nil
}
