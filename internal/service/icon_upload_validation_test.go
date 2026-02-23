package service

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/shiroha/subdux/internal/model"
)

func TestSanitizeUploadedIconRejectsExtensionMismatch(t *testing.T) {
	pngData := mustEncodePNG(t, 16, 16)

	_, _, err := sanitizeUploadedIcon(bytes.NewReader(pngData), "logo.ico", 65536)
	if err == nil {
		t.Fatal("sanitizeUploadedIcon() expected mismatch error")
	}
	if !errors.Is(err, ErrIconUploadContentMismatch) {
		t.Fatalf("sanitizeUploadedIcon() error = %v, want %v", err, ErrIconUploadContentMismatch)
	}
}

func TestSanitizeUploadedIconAcceptsICOAndStripsTrailingPayload(t *testing.T) {
	icoWithPayload := append(mustEncodeICOWithPNG(t, 32, 32), []byte("smuggled-payload")...)

	sanitized, ext, err := sanitizeUploadedIcon(bytes.NewReader(icoWithPayload), "logo.ico", 65536)
	if err != nil {
		t.Fatalf("sanitizeUploadedIcon() error = %v", err)
	}
	if ext != ".ico" {
		t.Fatalf("sanitizeUploadedIcon() ext = %q, want %q", ext, ".ico")
	}
	if strings.Contains(string(sanitized), "smuggled-payload") {
		t.Fatal("sanitizeUploadedIcon() should strip trailing payload bytes")
	}
	if !hasICOSignature(sanitized) {
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

	_, _, err := sanitizeUploadedIcon(bytes.NewReader(invalidICO), "logo.ico", 65536)
	if err == nil {
		t.Fatal("sanitizeUploadedIcon() expected invalid ico error")
	}
	if !errors.Is(err, ErrIconUploadInvalidICO) {
		t.Fatalf("sanitizeUploadedIcon() error = %v, want %v", err, ErrIconUploadInvalidICO)
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

	svc := NewSubscriptionService(db)
	iconValue, err := svc.UploadSubscriptionIcon(user.ID, sub.ID, bytes.NewReader(mustEncodeICOWithPNG(t, 32, 32)), "demo.ico", 65536)
	if err != nil {
		t.Fatalf("UploadSubscriptionIcon() error = %v", err)
	}
	if !strings.HasPrefix(iconValue, "file:") || !strings.HasSuffix(iconValue, ".ico") {
		t.Fatalf("UploadSubscriptionIcon() icon = %q, want managed .ico path", iconValue)
	}

	iconPath, ok := managedIconFilePath(iconValue)
	if !ok {
		t.Fatalf("managedIconFilePath(%q) should be valid", iconValue)
	}
	saved, err := os.ReadFile(iconPath)
	if err != nil {
		t.Fatalf("failed to read saved icon: %v", err)
	}
	if !hasICOSignature(saved) {
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

	svc := NewPaymentMethodService(db)
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

	svc := NewSubscriptionService(db)
	_, err := svc.UploadSubscriptionIcon(user.ID, sub.ID, bytes.NewReader(mustEncodePNG(t, 16, 16)), "demo.png", 65536)
	if !errors.Is(err, ErrImageUploadDisabled) {
		t.Fatalf("UploadSubscriptionIcon() error = %v, want %v", err, ErrImageUploadDisabled)
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

	svc := NewPaymentMethodService(db)
	_, err := svc.UploadPaymentMethodIcon(user.ID, method.ID, bytes.NewReader(mustEncodePNG(t, 16, 16)), "card.png", 65536)
	if !errors.Is(err, ErrImageUploadDisabled) {
		t.Fatalf("UploadPaymentMethodIcon() error = %v, want %v", err, ErrImageUploadDisabled)
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

func mustEncodeICOWithPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	pngData := mustEncodePNG(t, width, height)
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

	sanitized, ext, err := sanitizeUploadedIcon(bytes.NewReader(jpegBytes.Bytes()), "photo.jpeg", 65536)
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
