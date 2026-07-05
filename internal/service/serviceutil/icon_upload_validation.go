package serviceutil

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"path/filepath"
	"strings"

	"github.com/kasuha07/subdux/internal/service/serviceerr"
)

const (
	iconUploadUnsupportedTypeError = "only PNG, JPG, and ICO images are supported"
	iconUploadSizeLimitError       = "file size exceeds limit"
	iconUploadContentMismatchError = "icon file content does not match file extension"
	iconUploadInvalidICOError      = "ICO file must contain at least one valid PNG image"

	maxIconImageDimension = 2048
	maxIconImagePixels    = 1024 * 1024
)

// Icon-upload validation errors are typed so the transport layer maps them to a
// status without message matching: the four validation failures are 400
// (KindInvalid) and the administrator-disabled case is 403 (KindForbidden).
var (
	ErrIconUploadUnsupportedType = serviceerr.New(serviceerr.KindInvalid, iconUploadUnsupportedTypeError)
	ErrIconUploadSizeLimit       = serviceerr.New(serviceerr.KindInvalid, iconUploadSizeLimitError)
	ErrIconUploadContentMismatch = serviceerr.New(serviceerr.KindInvalid, iconUploadContentMismatchError)
	ErrIconUploadInvalidICO      = serviceerr.New(serviceerr.KindInvalid, iconUploadInvalidICOError)
	ErrImageUploadDisabled       = serviceerr.New(serviceerr.KindForbidden, "image uploads are disabled by administrator")
)

// SanitizeIconFile validates an icon file and returns a re-encoded safe image.
func SanitizeIconFile(file io.Reader, filename string, maxSize int64) ([]byte, string, error) {
	return SanitizeUploadedIcon(file, filename, maxSize)
}

type iconUploadFormat string

const (
	iconUploadFormatPNG iconUploadFormat = "png"
	iconUploadFormatJPG iconUploadFormat = "jpg"
	iconUploadFormatICO iconUploadFormat = "ico"
)

func SanitizeUploadedIcon(file io.Reader, filename string, maxSize int64) ([]byte, string, error) {
	ext := normalizeIconExtension(filename)
	if ext == "" {
		return nil, "", ErrIconUploadUnsupportedType
	}

	buf, err := io.ReadAll(io.LimitReader(file, maxSize+1))
	if err != nil {
		return nil, "", errors.New("failed to read file")
	}
	if int64(len(buf)) > maxSize {
		return nil, "", ErrIconUploadSizeLimit
	}

	sanitized, format, err := sanitizeIconContent(buf)
	if err != nil {
		return nil, "", err
	}
	if !iconExtensionMatchesFormat(ext, format) {
		return nil, "", ErrIconUploadContentMismatch
	}
	if int64(len(sanitized)) > maxSize {
		return nil, "", ErrIconUploadSizeLimit
	}

	return sanitized, normalizeSanitizedIconExtension(ext), nil
}

func normalizeIconExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".ico":
		return ext
	default:
		return ""
	}
}

func sanitizeIconContent(buf []byte) ([]byte, iconUploadFormat, error) {
	switch {
	case hasPNGSignature(buf):
		if err := validatePNGIconConfig(buf); err != nil {
			return nil, "", err
		}
		img, err := png.Decode(bytes.NewReader(buf))
		if err != nil {
			return nil, "", ErrIconUploadUnsupportedType
		}
		var out bytes.Buffer
		if err := png.Encode(&out, img); err != nil {
			return nil, "", errors.New("failed to sanitize icon")
		}
		return out.Bytes(), iconUploadFormatPNG, nil
	case hasJPEGSignature(buf):
		if err := validateJPEGIconConfig(buf); err != nil {
			return nil, "", err
		}
		img, err := jpeg.Decode(bytes.NewReader(buf))
		if err != nil {
			return nil, "", ErrIconUploadUnsupportedType
		}
		var out bytes.Buffer
		if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: 90}); err != nil {
			return nil, "", errors.New("failed to sanitize icon")
		}
		return out.Bytes(), iconUploadFormatJPG, nil
	case HasICOSignature(buf):
		sanitized, err := sanitizeICO(buf)
		if err != nil {
			return nil, "", err
		}
		return sanitized, iconUploadFormatICO, nil
	default:
		return nil, "", ErrIconUploadUnsupportedType
	}
}

func validatePNGIconConfig(buf []byte) error {
	cfg, err := png.DecodeConfig(bytes.NewReader(buf))
	if err != nil {
		return ErrIconUploadUnsupportedType
	}
	if !iconConfigWithinPixelLimit(cfg) {
		return ErrIconUploadSizeLimit
	}
	return nil
}

func validateJPEGIconConfig(buf []byte) error {
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(buf))
	if err != nil {
		return ErrIconUploadUnsupportedType
	}
	if !iconConfigWithinPixelLimit(cfg) {
		return ErrIconUploadSizeLimit
	}
	return nil
}

func iconConfigWithinPixelLimit(cfg image.Config) bool {
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return false
	}
	if cfg.Width > maxIconImageDimension || cfg.Height > maxIconImageDimension {
		return false
	}
	return cfg.Width <= maxIconImagePixels/cfg.Height
}

func normalizeSanitizedIconExtension(ext string) string {
	if ext == ".jpeg" {
		return ".jpg"
	}
	return ext
}

func hasPNGSignature(buf []byte) bool {
	return len(buf) >= 8 &&
		buf[0] == 0x89 &&
		buf[1] == 0x50 &&
		buf[2] == 0x4E &&
		buf[3] == 0x47 &&
		buf[4] == 0x0D &&
		buf[5] == 0x0A &&
		buf[6] == 0x1A &&
		buf[7] == 0x0A
}

func hasJPEGSignature(buf []byte) bool {
	return len(buf) >= 3 &&
		buf[0] == 0xFF &&
		buf[1] == 0xD8 &&
		buf[2] == 0xFF
}

func HasICOSignature(buf []byte) bool {
	return len(buf) >= 6 &&
		buf[0] == 0x00 &&
		buf[1] == 0x00 &&
		buf[2] == 0x01 &&
		buf[3] == 0x00
}

func iconExtensionMatchesFormat(ext string, format iconUploadFormat) bool {
	switch format {
	case iconUploadFormatPNG:
		return ext == ".png"
	case iconUploadFormatJPG:
		return ext == ".jpg" || ext == ".jpeg"
	case iconUploadFormatICO:
		return ext == ".ico"
	default:
		return false
	}
}

func sanitizeICO(buf []byte) ([]byte, error) {
	count := int(binary.LittleEndian.Uint16(buf[4:6]))
	if count <= 0 {
		return nil, ErrIconUploadInvalidICO
	}

	dirLen := 6 + count*16
	if len(buf) < dirLen {
		return nil, ErrIconUploadInvalidICO
	}

	type icoEntry struct {
		width  uint8
		height uint8
		data   []byte
	}

	validEntries := make([]icoEntry, 0, count)
	for i := 0; i < count; i++ {
		entryOffset := 6 + i*16
		size := int(binary.LittleEndian.Uint32(buf[entryOffset+8 : entryOffset+12]))
		dataOffset := int(binary.LittleEndian.Uint32(buf[entryOffset+12 : entryOffset+16]))
		if size <= 0 || dataOffset < dirLen || dataOffset+size > len(buf) {
			continue
		}

		data := buf[dataOffset : dataOffset+size]
		if !hasPNGSignature(data) {
			continue
		}
		cfg, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil || !iconConfigWithinPixelLimit(cfg) {
			continue
		}

		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			continue
		}

		var cleaned bytes.Buffer
		if err := png.Encode(&cleaned, img); err != nil {
			continue
		}

		width := img.Bounds().Dx()
		height := img.Bounds().Dy()
		if width <= 0 || height <= 0 {
			continue
		}

		validEntries = append(validEntries, icoEntry{
			width:  icoDimensionByte(width),
			height: icoDimensionByte(height),
			data:   bytes.Clone(cleaned.Bytes()),
		})
	}

	if len(validEntries) == 0 {
		return nil, ErrIconUploadInvalidICO
	}
	if len(validEntries) > math.MaxUint16 {
		return nil, ErrIconUploadInvalidICO
	}

	var out bytes.Buffer
	header := make([]byte, 6)
	binary.LittleEndian.PutUint16(header[2:4], 1)
	binary.LittleEndian.PutUint16(header[4:6], uint16(len(validEntries))) // #nosec G115 -- bounded by math.MaxUint16 above.
	out.Write(header)

	dataOffset := 6 + len(validEntries)*16
	entry := make([]byte, 16)
	for _, icoEntry := range validEntries {
		for i := range entry {
			entry[i] = 0
		}
		entry[0] = icoEntry.width
		entry[1] = icoEntry.height
		binary.LittleEndian.PutUint16(entry[4:6], 1)
		binary.LittleEndian.PutUint16(entry[6:8], 32)
		dataSize, ok := uint32FromBoundedInt(len(icoEntry.data))
		if !ok {
			return nil, ErrIconUploadInvalidICO
		}
		entryOffset, ok := uint32FromBoundedInt(dataOffset)
		if !ok {
			return nil, ErrIconUploadInvalidICO
		}
		binary.LittleEndian.PutUint32(entry[8:12], dataSize)
		binary.LittleEndian.PutUint32(entry[12:16], entryOffset)
		out.Write(entry)
		dataOffset += len(icoEntry.data)
	}

	for _, icoEntry := range validEntries {
		out.Write(icoEntry.data)
	}

	return out.Bytes(), nil
}

func icoDimensionByte(size int) uint8 {
	if size >= 256 {
		return 0
	}
	return uint8(size) // #nosec G115 -- caller rejects non-positive sizes and this branch is <= 255.
}

func uint32FromBoundedInt(value int) (uint32, bool) {
	if value < 0 || int64(value) > int64(math.MaxUint32) {
		return 0, false
	}
	return uint32(value), true // #nosec G115 -- value is explicitly bounded to uint32 range above.
}
