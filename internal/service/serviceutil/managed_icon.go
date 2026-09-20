package serviceutil

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"gorm.io/gorm"
)

// Upload endpoints generate these names; arbitrary legacy names have no proven owner.
var ownedIconName = regexp.MustCompile(`^[1-9][0-9]*_(payment_)?[1-9][0-9]*_[0-9]+\.(png|jpg|jpeg|ico)$`)

func ownedIconPath(userID uint, icon string) (string, bool) {
	name, ok := strings.CutPrefix(icon, "file:")
	if !ok || !ownedIconName.MatchString(name) || !strings.HasPrefix(name, fmt.Sprintf("%d_", userID)) {
		return "", false
	}
	return filepath.Join(pkg.GetDataPath(), "assets", "icons", name), true
}

// ValidateManagedIconOwnership applies at service boundaries, including imports.
// A filename supplied by a client is never itself authority to use another user's asset.
func ValidateManagedIconOwnership(userID uint, icon string) error {
	if !strings.HasPrefix(icon, "file:") {
		return nil
	}
	if _, ok := ownedIconPath(userID, icon); !ok {
		return serviceerr.New(serviceerr.KindInvalid, "invalid_icon_value", "invalid icon value")
	}
	return nil
}

// RemoveUnreferencedManagedIcon is deliberately conservative for legacy data and
// failed queries. References in either domain, including other users, retain the file.
func RemoveUnreferencedManagedIcon(db *gorm.DB, userID uint, icon string) {
	path, ok := ownedIconPath(userID, icon)
	if !ok {
		return
	}
	for _, record := range []interface{}{&model.Subscription{}, &model.PaymentMethod{}} {
		var count int64
		if err := db.Model(record).Where("icon = ?", icon).Count(&count).Error; err != nil || count > 0 {
			return
		}
	}
	_ = os.Remove(path)
}
