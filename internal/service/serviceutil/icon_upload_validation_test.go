package serviceutil_test

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	catalogservice "github.com/kasuha07/subdux/internal/service/catalog"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
)

func TestSanitizeUploadedIconRejectsExtensionMismatch(t *testing.T) {
	pngData := mustEncodePNG(t, 16, 16)

	_, _, err := serviceutil.SanitizeUploadedIcon(bytes.NewReader(pngData), "logo.ico", 65536)
	if err == nil {
		t.Fatal("sanitizeUploadedIcon() expected mismatch error")
	}
	if !errors.Is(err, serviceutil.ErrIconUploadContentMismatch) {
		t.Fatalf("sanitizeUploadedIcon() error = %v, want %v", err, serviceutil.ErrIconUploadContentMismatch)
	}
}

func TestSanitizeUploadedIconAcceptsICOAndStripsTrailingPayload(t *testing.T) {
	icoWithPayload := append(mustEncodeICOWithPNG(t, 32, 32), []byte("smuggled-payload")...)

	sanitized, ext, err := serviceutil.SanitizeUploadedIcon(bytes.NewReader(icoWithPayload), "logo.ico", 65536)
	if err != nil {
		t.Fatalf("sanitizeUploadedIcon() error = %v", err)
	}
	if ext != ".ico" {
		t.Fatalf("sanitizeUploadedIcon() ext = %q, want %q", ext, ".ico")
	}
	if strings.Contains(string(sanitized), "smuggled-payload") {
		t.Fatal("sanitizeUploadedIcon() should strip trailing payload bytes")
	}
	if !serviceutil.HasICOSignature(sanitized) {
		t.Fatal("sanitizeUploadedIcon() should preserve ico container format")
	}
}

func TestSanitizeUploadedIconRejectsICOWithoutPNGImage(t *testing.T) {
	invalidICO := make([]byte, 6+16+8)
	binary.LittleEndian.PutUint16(invalidICO[2:4], 1)
	binary.LittleEndian.PutUint16(invalidICO[4:6], 1)
	binary.LittleEndian.PutUint32(invalidICO[14:18], 8)
	binary.LittleEndian.PutUint32(invalidICO[18:22], 22)
	copy(invalidICO[22:], []byte("notapng!"))

	_, _, err := serviceutil.SanitizeUploadedIcon(bytes.NewReader(invalidICO), "logo.ico", 65536)
	if err == nil {
		t.Fatal("sanitizeUploadedIcon() expected invalid ico error")
	}
	if !errors.Is(err, serviceutil.ErrIconUploadInvalidICO) {
		t.Fatalf("sanitizeUploadedIcon() error = %v, want %v", err, serviceutil.ErrIconUploadInvalidICO)
	}
}

func TestSanitizeUploadedIconRejectsCompressedPixelBombPNGBeforeDecode(t *testing.T) {
	pngData := mustEncodeZeroRGBAPNG(t, 4096, 4096)
	if len(pngData) > 65536 {
		t.Fatalf("test png size = %d, want <= 65536", len(pngData))
	}

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	_, _, err := serviceutil.SanitizeUploadedIcon(bytes.NewReader(pngData), "large.png", 65536)
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	if !errors.Is(err, serviceutil.ErrIconUploadSizeLimit) {
		t.Fatalf("sanitizeUploadedIcon() error = %v, want %v", err, serviceutil.ErrIconUploadSizeLimit)
	}
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 16<<20 {
		t.Fatalf("sanitizeUploadedIcon() allocated %d bytes for rejected PNG, want <= %d", allocated, 16<<20)
	}
}

func TestSanitizeUploadedIconRejectsOversizedJPEGDimensions(t *testing.T) {
	jpegData := mustEncodeJPEG(t, 1025, 1025)
	if len(jpegData) > 65536 {
		t.Fatalf("test jpeg size = %d, want <= 65536", len(jpegData))
	}

	_, _, err := serviceutil.SanitizeUploadedIcon(bytes.NewReader(jpegData), "large.jpg", 65536)
	if !errors.Is(err, serviceutil.ErrIconUploadSizeLimit) {
		t.Fatalf("sanitizeUploadedIcon() error = %v, want %v", err, serviceutil.ErrIconUploadSizeLimit)
	}
}

func TestSanitizeUploadedIconRejectsOversizedICOPNGDimensions(t *testing.T) {
	pngData := mustEncodeZeroRGBAPNG(t, 1025, 1025)
	icoData := mustEncodeICOContainer(t, 1025, 1025, pngData)
	if len(icoData) > 65536 {
		t.Fatalf("test ico size = %d, want <= 65536", len(icoData))
	}

	_, _, err := serviceutil.SanitizeUploadedIcon(bytes.NewReader(icoData), "large.ico", 65536)
	if !errors.Is(err, serviceutil.ErrIconUploadInvalidICO) {
		t.Fatalf("sanitizeUploadedIcon() error = %v, want %v", err, serviceutil.ErrIconUploadInvalidICO)
	}
}

func TestUploadSubscriptionIconAcceptsICO(t *testing.T) {
	dataPath := t.TempDir()
	t.Setenv("DATA_PATH", dataPath)

	db := newTestDB(t)
	user := createTestUser(t, db)
	sub := model.Subscription{
		UserID:   user.ID,
		Name:     "Demo",
		Amount:   1.99,
		Currency: "USD",
	}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}

	svc := subscriptionservice.NewService(db)
	iconValue, err := svc.UploadSubscriptionIcon(user.ID, sub.ID, bytes.NewReader(mustEncodeICOWithPNG(t, 32, 32)), "demo.ico", 65536)
	if err != nil {
		t.Fatalf("UploadSubscriptionIcon() error = %v", err)
	}
	if !strings.HasPrefix(iconValue, "file:") || !strings.HasSuffix(iconValue, ".ico") {
		t.Fatalf("UploadSubscriptionIcon() icon = %q, want managed .ico path", iconValue)
	}

	iconPath, ok := subscriptionservice.ManagedIconFilePath(iconValue)
	if !ok {
		t.Fatalf("managedIconFilePath(%q) should be valid", iconValue)
	}
	saved, err := os.ReadFile(iconPath)
	if err != nil {
		t.Fatalf("failed to read saved icon: %v", err)
	}
	if !serviceutil.HasICOSignature(saved) {
		t.Fatal("saved icon should be ico")
	}
}

func TestUploadPaymentMethodIconAcceptsICO(t *testing.T) {
	dataPath := t.TempDir()
	t.Setenv("DATA_PATH", dataPath)

	db := newTestDB(t)
	user := createTestUser(t, db)
	method := model.PaymentMethod{
		UserID:         user.ID,
		Name:           "Card",
		NameCustomized: true,
		Icon:           "",
	}
	if err := db.Create(&method).Error; err != nil {
		t.Fatalf("failed to create payment method: %v", err)
	}

	svc := catalogservice.NewPaymentMethodService(db)
	iconValue, err := svc.UploadPaymentMethodIcon(user.ID, method.ID, bytes.NewReader(mustEncodeICOWithPNG(t, 24, 24)), "card.ico", 65536)
	if err != nil {
		t.Fatalf("UploadPaymentMethodIcon() error = %v", err)
	}
	if !strings.HasPrefix(iconValue, "file:") || !strings.HasSuffix(iconValue, ".ico") {
		t.Fatalf("UploadPaymentMethodIcon() icon = %q, want managed .ico path", iconValue)
	}
}

func TestUploadSubscriptionIconBlockedWhenImageUploadDisabled(t *testing.T) {
	dataPath := t.TempDir()
	t.Setenv("DATA_PATH", dataPath)

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings: %v", err)
	}
	user := createTestUser(t, db)
	sub := model.Subscription{
		UserID:   user.ID,
		Name:     "Demo",
		Amount:   1.99,
		Currency: "USD",
	}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "allow_image_upload", Value: "false"}).Error; err != nil {
		t.Fatalf("failed to disable image upload: %v", err)
	}

	svc := subscriptionservice.NewService(db)
	_, err := svc.UploadSubscriptionIcon(user.ID, sub.ID, bytes.NewReader(mustEncodePNG(t, 16, 16)), "demo.png", 65536)
	if !errors.Is(err, serviceutil.ErrImageUploadDisabled) {
		t.Fatalf("UploadSubscriptionIcon() error = %v, want %v", err, serviceutil.ErrImageUploadDisabled)
	}
}

func TestUploadPaymentMethodIconBlockedWhenImageUploadDisabled(t *testing.T) {
	dataPath := t.TempDir()
	t.Setenv("DATA_PATH", dataPath)

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings: %v", err)
	}
	user := createTestUser(t, db)
	method := model.PaymentMethod{
		UserID:         user.ID,
		Name:           "Card",
		NameCustomized: true,
		Icon:           "",
	}
	if err := db.Create(&method).Error; err != nil {
		t.Fatalf("failed to create payment method: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "allow_image_upload", Value: "false"}).Error; err != nil {
		t.Fatalf("failed to disable image upload: %v", err)
	}

	svc := catalogservice.NewPaymentMethodService(db)
	_, err := svc.UploadPaymentMethodIcon(user.ID, method.ID, bytes.NewReader(mustEncodePNG(t, 16, 16)), "card.png", 65536)
	if !errors.Is(err, serviceutil.ErrImageUploadDisabled) {
		t.Fatalf("UploadPaymentMethodIcon() error = %v, want %v", err, serviceutil.ErrImageUploadDisabled)
	}
}

func mustEncodePNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.NRGBA{R: 32, G: 140, B: 230, A: 255})
		}
	}

	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatalf("failed to encode png: %v", err)
	}
	return out.Bytes()
}

func mustEncodeJPEG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, width, height))
	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: 75}); err != nil {
		t.Fatalf("failed to encode jpeg: %v", err)
	}
	return out.Bytes()
}

func mustEncodeZeroRGBAPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	var idat bytes.Buffer
	zw, err := zlib.NewWriterLevel(&idat, zlib.BestCompression)
	if err != nil {
		t.Fatalf("failed to create zlib writer: %v", err)
	}
	row := make([]byte, 1+width*4)
	for y := 0; y < height; y++ {
		if _, err := zw.Write(row); err != nil {
			t.Fatalf("failed to write png row: %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zlib writer: %v", err)
	}

	var out bytes.Buffer
	out.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	var ihdr [13]byte
	binary.BigEndian.PutUint32(ihdr[0:4], uint32(width))
	binary.BigEndian.PutUint32(ihdr[4:8], uint32(height))
	ihdr[8] = 8
	ihdr[9] = 6
	writePNGChunk(&out, "IHDR", ihdr[:])
	writePNGChunk(&out, "IDAT", idat.Bytes())
	writePNGChunk(&out, "IEND", nil)
	return out.Bytes()
}

func writePNGChunk(out *bytes.Buffer, chunkType string, data []byte) {
	var scratch [4]byte
	binary.BigEndian.PutUint32(scratch[:], uint32(len(data)))
	out.Write(scratch[:])
	out.WriteString(chunkType)
	out.Write(data)

	crc := crc32.NewIEEE()
	_, _ = crc.Write([]byte(chunkType))
	_, _ = crc.Write(data)
	binary.BigEndian.PutUint32(scratch[:], crc.Sum32())
	out.Write(scratch[:])
}

func mustEncodeICOWithPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	pngData := mustEncodePNG(t, width, height)
	return mustEncodeICOContainer(t, width, height, pngData)
}

func mustEncodeICOContainer(t *testing.T, width, height int, pngData []byte) []byte {
	t.Helper()
	header := make([]byte, 6)
	binary.LittleEndian.PutUint16(header[2:4], 1)
	binary.LittleEndian.PutUint16(header[4:6], 1)

	entry := make([]byte, 16)
	if width >= 256 {
		entry[0] = 0
	} else {
		entry[0] = uint8(width)
	}
	if height >= 256 {
		entry[1] = 0
	} else {
		entry[1] = uint8(height)
	}
	binary.LittleEndian.PutUint16(entry[4:6], 1)
	binary.LittleEndian.PutUint16(entry[6:8], 32)
	binary.LittleEndian.PutUint32(entry[8:12], uint32(len(pngData)))
	binary.LittleEndian.PutUint32(entry[12:16], 22)

	var out bytes.Buffer
	out.Write(header)
	out.Write(entry)
	out.Write(pngData)
	return out.Bytes()
}

func TestSanitizeUploadedIconSanitizesJPEG(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	img.Set(0, 0, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	var jpegBytes bytes.Buffer
	if err := jpeg.Encode(&jpegBytes, img, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatalf("failed to encode jpeg: %v", err)
	}

	sanitized, ext, err := serviceutil.SanitizeUploadedIcon(bytes.NewReader(jpegBytes.Bytes()), "photo.jpeg", 65536)
	if err != nil {
		t.Fatalf("sanitizeUploadedIcon() error = %v", err)
	}
	if ext != ".jpg" {
		t.Fatalf("sanitizeUploadedIcon() ext = %q, want %q", ext, ".jpg")
	}

	_, format, err := image.DecodeConfig(bytes.NewReader(sanitized))
	if err != nil {
		t.Fatalf("failed to decode sanitized jpeg: %v", err)
	}
	if format != "jpeg" {
		t.Fatalf("sanitized format = %q, want jpeg", format)
	}
}
