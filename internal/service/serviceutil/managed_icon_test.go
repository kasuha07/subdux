package serviceutil_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	catalog "github.com/kasuha07/subdux/internal/service/catalog"
	importer "github.com/kasuha07/subdux/internal/service/importer"
	"github.com/kasuha07/subdux/internal/service/servicetest"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	subscription "github.com/kasuha07/subdux/internal/service/subscription"
)

func TestManagedIconOwnershipAndReferenceCleanup(t *testing.T) {
	t.Setenv("DATA_PATH", t.TempDir())
	db := servicetest.NewDB(t)
	victim := servicetest.CreateUser(t, db)
	attacker := model.User{Username: "attacker", Email: "attacker@example.com", Password: "unused", Status: "active"}
	if err := db.Create(&attacker).Error; err != nil {
		t.Fatal(err)
	}
	icon := fmt.Sprintf("file:%d_1_123.png", victim.ID)
	path, ok := subscription.ManagedIconFilePath(icon)
	if !ok {
		t.Fatal("bad test icon")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("test image"), 0600); err != nil {
		t.Fatal(err)
	}
	method := model.PaymentMethod{UserID: victim.ID, Name: "Victim", Icon: icon}
	if err := db.Create(&method).Error; err != nil {
		t.Fatal(err)
	}
	methods := catalog.NewPaymentMethodService(db)
	subs := subscription.NewService(db)
	if _, err := methods.Create(attacker.ID, catalog.CreatePaymentMethodInput{Name: "Attack", Icon: icon}); err == nil {
		t.Fatal("foreign payment icon accepted")
	}
	if _, err := subs.Create(attacker.ID, subscription.CreateSubscriptionInput{Name: "Attack", Icon: icon}); err == nil {
		t.Fatal("foreign subscription icon accepted")
	}
	ownMethod, err := methods.Create(attacker.ID, catalog.CreatePaymentMethodInput{Name: "Own"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := methods.Update(attacker.ID, ownMethod.ID, catalog.UpdatePaymentMethodInput{Icon: &icon}); err == nil {
		t.Fatal("foreign payment icon update accepted")
	}
	ownSub := model.Subscription{UserID: attacker.ID, Name: "Own", Amount: 1, Currency: "USD", BillingType: "one_time"}
	if err := db.Create(&ownSub).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := subs.Update(attacker.ID, ownSub.ID, subscription.UpdateSubscriptionInput{Icon: &icon}); err == nil {
		t.Fatal("foreign subscription icon update accepted")
	}

	// Imported records must not bypass service ownership checks.
	result, err := importer.NewService(db).ImportFromSubdux(attacker.ID, importer.SubduxImportData{
		PaymentMethods: []model.PaymentMethod{{Name: "Imported attack", Icon: icon}},
		Currencies:     []model.UserCurrency{}, Categories: []model.Category{}, Subscriptions: []model.Subscription{},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Result.Errors) == 0 {
		t.Fatal("foreign imported icon accepted")
	}

	// Legacy malicious references must not confer deletion authority.
	legacy := model.PaymentMethod{UserID: attacker.ID, Name: "Legacy attack", Icon: icon}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	if err := methods.Delete(attacker.ID, legacy.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("victim file removed: %v", err)
	}

	// Sharing across domains is retained until the last reference is gone.
	shared := model.Subscription{UserID: victim.ID, Name: "Shared", Icon: icon, Amount: 1, Currency: "USD", BillingType: "one_time"}
	if err := db.Create(&shared).Error; err != nil {
		t.Fatal(err)
	}
	if err := methods.Delete(victim.ID, method.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("shared file removed: %v", err)
	}
	if err := subs.Delete(victim.ID, shared.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("unreferenced file remains: %v", err)
	}
}

func TestManagedIconRejectsUnprovenOwnership(t *testing.T) {
	for _, icon := range []string{"file:other.png", "file:10_1_1.png", "file:1_../../x.png", "file:1_payment_2_1.svg"} {
		if err := serviceutil.ValidateManagedIconOwnership(1, icon); err == nil {
			t.Fatalf("accepted %q", icon)
		}
	}
	for _, icon := range []string{"", "💳", "file:1_payment_2_123.png", "file:1_2_123.ico"} {
		if err := serviceutil.ValidateManagedIconOwnership(1, icon); err != nil {
			t.Fatalf("rejected %q: %v", icon, err)
		}
	}
}
