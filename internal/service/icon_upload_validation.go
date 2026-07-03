package service

import (
	"io"

	"github.com/kasuha07/subdux/internal/service/serviceutil"
)

var (
	ErrIconUploadUnsupportedType = serviceutil.ErrIconUploadUnsupportedType
	ErrIconUploadSizeLimit       = serviceutil.ErrIconUploadSizeLimit
	ErrIconUploadContentMismatch = serviceutil.ErrIconUploadContentMismatch
	ErrIconUploadInvalidICO      = serviceutil.ErrIconUploadInvalidICO
)

// SanitizeIconFile validates an icon file and returns a re-encoded safe image.
func SanitizeIconFile(file io.Reader, filename string, maxSize int64) ([]byte, string, error) {
	return serviceutil.SanitizeIconFile(file, filename, maxSize)
}

func sanitizeUploadedIcon(file io.Reader, filename string, maxSize int64) ([]byte, string, error) {
	return serviceutil.SanitizeUploadedIcon(file, filename, maxSize)
}

func hasICOSignature(buf []byte) bool {
	return serviceutil.HasICOSignature(buf)
}
