package backup

import (
	"context"
	"net/http"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/outbound"
	"gorm.io/gorm"
)

// newBackupTarget resolves a persisted destination row into a live target. The
// row's encrypted config is decrypted and parsed here so each target
// constructor receives a plain config map. Unknown types are rejected rather
// than silently skipped, so a misconfigured row surfaces as an error during a
// run instead of being quietly dropped.
//
// db is threaded through for targets that need outbound-policy-aware HTTP
// clients (S3, WebDAV); the local target ignores it.
func newBackupTarget(destination model.BackupDestination, db *gorm.DB) (BackupTarget, error) {
	plain, err := decryptDestinationConfig(destination.Config)
	if err != nil {
		return nil, err
	}
	config, err := parseDestinationConfigMap(plain)
	if err != nil {
		return nil, err
	}
	return newBackupTargetFromConfig(destination.Type, config, db)
}

// newBackupTargetFromConfig builds a target from an already-decrypted config
// map. It is the shared construction point for both runtime delivery (via
// newBackupTarget) and config validation (which builds a throwaway target to
// exercise type-specific invariants without persisting).
func newBackupTargetFromConfig(destinationType string, config map[string]any, db *gorm.DB) (BackupTarget, error) {
	switch destinationType {
	case "local":
		return newLocalTarget(config)
	case "s3":
		return newS3Target(config, db)
	case "webdav":
		return newWebDAVTarget(config, db)
	default:
		return nil, ErrUnknownBackupDestinationType
	}
}

// newBackupDestinationHTTPClient builds the HTTP client used by the remote
// backup transports (S3, WebDAV). Going through the purpose-aware constructor is
// what actually binds these transports to PurposeBackupDestination in
// outbound.PurposeAppliesSSRFPolicy: a backup endpoint is chosen by an
// administrator, so it is deliberately exempt from the user-facing SSRF filter
// (an admin may target a private-network MinIO or Nextcloud) and relies on the
// configured proxy and network ACL boundary instead. Building the client here
// rather than calling NewOutboundHTTPClient directly keeps that trust-boundary
// decision discoverable from the policy map instead of only from a comment.
//
// A nil db (unit contexts) or a construction failure falls back to the plain
// proxy-aware client, which is the same transport this purpose resolves to.
func newBackupDestinationHTTPClient(db *gorm.DB, timeout time.Duration) *http.Client {
	client, err := outbound.BuildHTTPClientWithTimeout(context.Background(), db, outbound.PurposeBackupDestination, timeout)
	if err != nil {
		return outbound.NewOutboundHTTPClient(db, timeout)
	}
	return client
}
