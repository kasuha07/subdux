package serviceutil_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/admin"
	"github.com/kasuha07/subdux/internal/service/catalog"
	"github.com/kasuha07/subdux/internal/service/notification"
	"github.com/kasuha07/subdux/internal/service/servicetest"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	"github.com/kasuha07/subdux/internal/service/settings"
	"github.com/kasuha07/subdux/internal/service/subscription"
	"gorm.io/gorm"
)

func revisionDB(t *testing.T) (*gorm.DB, uint) {
	t.Helper()
	db := servicetest.NewDB(t)
	if err := db.AutoMigrate(&model.NotificationPolicy{}, &model.NotificationChannel{}, &model.NotificationOutbox{}); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db, servicetest.CreateUser(t, db).ID
}

func revisionPtr[T any](value T) *T { return &value }

func requireRevisionConflict(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, serviceutil.ErrRevisionConflict) {
		t.Fatalf("error = %v, want revision conflict", err)
	}
}

func TestCatalogConcurrentRevisionHasOneWinner(t *testing.T) {
	db, userID := revisionDB(t)
	svc := catalog.NewCategoryService(db)
	row, err := svc.Create(userID, catalog.CreateCategoryInput{Name: "original"})
	if err != nil {
		t.Fatal(err)
	}
	if row.Revision != 1 {
		t.Fatalf("initial revision = %d", row.Revision)
	}
	start := make(chan struct{})
	errorsCh := make(chan error, 2)
	var wg sync.WaitGroup
	for _, name := range []string{"first", "second"} {
		wg.Go(func() {
			<-start
			_, err := svc.Update(userID, row.ID, catalog.UpdateCategoryInput{Revision: row.Revision, Name: revisionPtr(name)})
			errorsCh <- err
		})
	}
	close(start)
	wg.Wait()
	close(errorsCh)
	winners, conflicts := 0, 0
	for err := range errorsCh {
		if err == nil {
			winners++
		} else if errors.Is(err, serviceutil.ErrRevisionConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if winners != 1 || conflicts != 1 {
		t.Fatalf("winners=%d conflicts=%d", winners, conflicts)
	}
	var saved model.Category
	if err := db.First(&saved, row.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Revision != 2 || saved.Name == "original" {
		t.Fatalf("saved = %+v", saved)
	}
	requireRevisionConflict(t, svc.Delete(userID, row.ID, row.Revision))
}

func TestCatalogRevisionReorderRollsBackAndInvalidatesOldEdits(t *testing.T) {
	db, userID := revisionDB(t)
	svc := catalog.NewCategoryService(db)
	a, err := svc.Create(userID, catalog.CreateCategoryInput{Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Create(userID, catalog.CreateCategoryInput{Name: "B"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Update(userID, b.ID, catalog.UpdateCategoryInput{Revision: b.Revision, Name: revisionPtr("B2")})
	if err != nil {
		t.Fatal(err)
	}
	requireRevisionConflict(t, svc.Reorder(userID, []catalog.ReorderItem{{ID: a.ID, Revision: 1, SortOrder: 5}, {ID: b.ID, Revision: 1, SortOrder: 6}}))
	var saved model.Category
	if err := db.First(&saved, a.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Revision != 1 || saved.DisplayOrder != 0 {
		t.Fatalf("partial reorder persisted: %+v", saved)
	}
	if err := svc.Reorder(userID, []catalog.ReorderItem{{ID: a.ID, Revision: 1, SortOrder: 5}}); err != nil {
		t.Fatal(err)
	}
	_, err = svc.Update(userID, a.ID, catalog.UpdateCategoryInput{Revision: 1, Name: revisionPtr("stale")})
	requireRevisionConflict(t, err)
}

func TestCurrencyAndPaymentMethodRejectStaleEdits(t *testing.T) {
	db, userID := revisionDB(t)
	currencies := catalog.NewCurrencyService(db)
	currency, err := currencies.Create(userID, catalog.CreateCurrencyInput{Code: "USD"})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := currencies.Update(userID, currency.ID, catalog.UpdateCurrencyInput{Revision: currency.Revision, Alias: revisionPtr("Dollar")})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != 2 {
		t.Fatalf("revision = %d", updated.Revision)
	}
	_, err = currencies.Update(userID, currency.ID, catalog.UpdateCurrencyInput{Revision: currency.Revision, Symbol: revisionPtr("$")})
	requireRevisionConflict(t, err)
	methods := catalog.NewPaymentMethodService(db)
	method, err := methods.Create(userID, catalog.CreatePaymentMethodInput{Name: "Card"})
	if err != nil {
		t.Fatal(err)
	}
	updatedMethod, err := methods.Update(userID, method.ID, catalog.UpdatePaymentMethodInput{Revision: method.Revision, Name: revisionPtr("Card2")})
	if err != nil {
		t.Fatal(err)
	}
	if updatedMethod.Revision != 2 {
		t.Fatalf("revision = %d", updatedMethod.Revision)
	}
	_, err = methods.Update(userID, method.ID, catalog.UpdatePaymentMethodInput{Revision: method.Revision, SortOrder: revisionPtr(8)})
	requireRevisionConflict(t, err)
	requireRevisionConflict(t, methods.Delete(userID, method.ID, method.Revision))
}

func TestSubscriptionRevisionProtectsRenewalAndEvents(t *testing.T) {
	db, userID := revisionDB(t)
	svc := subscription.NewService(db)
	row, err := svc.Create(userID, subscription.CreateSubscriptionInput{Name: "Plan", Amount: 10, Currency: "USD", Status: "active", RenewalMode: "manual_renew", BillingType: "recurring", RecurrenceType: "interval", IntervalCount: revisionPtr(1), IntervalUnit: "month", NextBillingDate: "2099-01-01"})
	if err != nil {
		t.Fatal(err)
	}
	ended, err := svc.Update(userID, row.ID, subscription.UpdateSubscriptionInput{Revision: row.Revision, Status: revisionPtr("ended")})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Update(userID, row.ID, subscription.UpdateSubscriptionInput{Revision: row.Revision, Name: revisionPtr("stale")})
	requireRevisionConflict(t, err)
	_, err = svc.MarkManualRenewed(userID, row.ID, row.Revision)
	if err == nil {
		t.Fatal("stale renewal succeeded")
	}
	requireRevisionConflict(t, svc.Delete(userID, row.ID, row.Revision))
	var saved model.Subscription
	if err := db.First(&saved, row.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Status != "ended" || saved.Revision != ended.Revision || saved.Name != "Plan" {
		t.Fatalf("saved = %+v", saved)
	}
	var count int64
	if err := db.Model(&model.SubscriptionEvent{}).Where("subscription_id = ?", row.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("event count = %d, want create and successful edit only", count)
	}
}

func TestNotificationRevisionsProtectPartialEdits(t *testing.T) {
	db, userID := revisionDB(t)
	svc := notification.NewService(db, nil, nil)
	policy, err := svc.UpdatePolicy(userID, notification.UpdatePolicyInput{DaysBefore: revisionPtr(3)})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := svc.UpdatePolicy(userID, notification.UpdatePolicyInput{Revision: revisionPtr(policy.Revision), DaysBefore: revisionPtr(5)})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.UpdatePolicy(userID, notification.UpdatePolicyInput{Revision: revisionPtr(policy.Revision), QuietHoursStart: revisionPtr("22:00")})
	requireRevisionConflict(t, err)
	if updated.Revision != policy.Revision+1 {
		t.Fatalf("revision = %d", updated.Revision)
	}
	channel := model.NotificationChannel{UserID: userID, Type: "smtp", Config: "{}"}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatal(err)
	}
	_, err = svc.UpdateChannel(userID, channel.ID, notification.UpdateChannelInput{Revision: channel.Revision, Enabled: revisionPtr(false)})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.UpdateChannel(userID, channel.ID, notification.UpdateChannelInput{Revision: channel.Revision, Enabled: revisionPtr(true)})
	requireRevisionConflict(t, err)
	requireRevisionConflict(t, svc.DeleteChannel(userID, channel.ID, channel.Revision))
	templates := notification.NewNotificationTemplateService(db, notification.NewTemplateValidator())
	tmpl, err := templates.CreateTemplate(userID, notification.CreateTemplateInput{Format: "plaintext", Template: "original"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = templates.UpdateTemplate(userID, tmpl.ID, notification.UpdateTemplateInput{Revision: tmpl.Revision, Template: revisionPtr("new")})
	if err != nil {
		t.Fatal(err)
	}
	_, err = templates.UpdateTemplate(userID, tmpl.ID, notification.UpdateTemplateInput{Revision: tmpl.Revision, Template: revisionPtr("old")})
	requireRevisionConflict(t, err)
	requireRevisionConflict(t, templates.DeleteTemplate(userID, tmpl.ID, tmpl.Revision))
}

func TestSystemSettingsRevisionSnapshotIsAtomic(t *testing.T) {
	db, _ := revisionDB(t)
	if err := settings.SaveString(db, "site_name", "original"); err != nil {
		t.Fatal(err)
	}
	svc := admin.NewService(db)
	snapshot, err := svc.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.UpdateSettings(admin.UpdateSettingsInput{Revisions: snapshot.Revisions, SiteName: revisionPtr("new")}); err != nil {
		t.Fatal(err)
	}
	requireRevisionConflict(t, svc.UpdateSettings(admin.UpdateSettingsInput{Revisions: snapshot.Revisions, SiteName: revisionPtr("stale"), SMTPHost: revisionPtr("should-not-save")}))
	fresh, err := svc.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if fresh.SiteName != "new" || fresh.SMTPHost != "" || fresh.Revisions["site_name"] != snapshot.Revisions["site_name"]+1 {
		t.Fatalf("unexpected persisted settings")
	}
	if err := svc.UpdateSettings(admin.UpdateSettingsInput{Revisions: fresh.Revisions, SiteName: revisionPtr("rollback"), SystemProxyType: revisionPtr("http"), SystemProxyURL: revisionPtr("socks5://proxy.example.com:1080")}); err == nil {
		t.Fatal("invalid settings accepted")
	}
	after, err := svc.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if after.SiteName != fresh.SiteName || after.Revisions["site_name"] != fresh.Revisions["site_name"] {
		t.Fatal("failed transaction changed value or revision")
	}
}
