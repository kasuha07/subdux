package backup

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/outbound"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"gorm.io/gorm"
)

// ErrBackupSkipTLSVerifyUnsupported is returned when a destination asks to skip
// certificate verification but the outbound transport resolved for it cannot
// carry that setting. Refusing to build the target is deliberate: the
// alternative is a destination that quietly keeps verifying and fails its
// handshake with no explanation of why the switch had no effect.
var ErrBackupSkipTLSVerifyUnsupported = serviceerr.New(
	serviceerr.KindInternal,
	"backup_skip_tls_verify_unsupported",
	"skip TLS verification is not supported by the configured outbound transport",
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
// skipTLSVerify is the per-destination compatibility switch for endpoints
// presenting a self-signed or otherwise unverifiable certificate, matching the
// SMTP sender's switch. It is scoped to the one destination that opted in: the
// transport is cloned rather than adjusted in place precisely so that scoping
// holds (see backupDestinationTransport).
//
// A nil db (unit contexts) or a construction failure falls back to the plain
// proxy-aware client, which is the same transport this purpose resolves to.
func newBackupDestinationHTTPClient(db *gorm.DB, timeout time.Duration, skipTLSVerify bool) (*http.Client, error) {
	client, err := outbound.BuildHTTPClientWithTimeout(context.Background(), db, outbound.PurposeBackupDestination, timeout)
	if err != nil {
		client = outbound.NewOutboundHTTPClient(db, timeout)
	}
	if !skipTLSVerify {
		return client, nil
	}

	transport, err := backupDestinationTransport(client.Transport)
	if err != nil {
		return nil, err
	}
	client.Transport = transport
	return client, nil
}

// backupDestinationTransport returns a copy of base with certificate
// verification disabled.
//
// Cloning is the whole point: for this purpose the outbound constructor hands
// back either http.DefaultTransport — shared by every outbound caller in the
// process — or a transport it clones for the proxy path, and disabling
// verification on the former in place would silently switch it off for OIDC,
// notifications, and the icon proxy too. The clone keeps the proxy and dialer
// configuration that was resolved for this purpose while confining the relaxed
// TLS policy to the destination that asked for it.
//
// A transport this function cannot copy is an error rather than a silent
// fallback to ordinary verification: an admin who enabled the switch for a
// self-signed endpoint would otherwise see nothing but unexplained handshake
// failures.
func backupDestinationTransport(base http.RoundTripper) (*http.Transport, error) {
	var transport *http.Transport
	switch typed := base.(type) {
	case nil:
		// A nil Transport means the client falls back to http.DefaultTransport.
		transport = defaultHTTPTransportClone()
	case *http.Transport:
		transport = typed.Clone()
	default:
		return nil, ErrBackupSkipTLSVerifyUnsupported
	}
	if transport == nil {
		return nil, ErrBackupSkipTLSVerifyUnsupported
	}

	tlsConfig := transport.TLSClientConfig
	if tlsConfig == nil {
		tlsConfig = &tls.Config{}
	} else {
		tlsConfig = tlsConfig.Clone()
	}
	if tlsConfig.MinVersion == 0 {
		// The switch is about an unverifiable certificate, not about accepting a
		// legacy protocol version. Pinning the floor that Go would apply anyway
		// makes that explicit so relaxing verification can never read as
		// permission to negotiate down.
		tlsConfig.MinVersion = tls.VersionTLS12
	}
	// #nosec G402 -- Not a security issue: default false; admin-only
	// compatibility switch for trusted self-signed backup endpoints, scoped to
	// the single destination that enabled it.
	tlsConfig.InsecureSkipVerify = true
	transport.TLSClientConfig = tlsConfig
	return transport, nil
}

func defaultHTTPTransportClone() *http.Transport {
	if transport, ok := http.DefaultTransport.(*http.Transport); ok {
		return transport.Clone()
	}
	return nil
}
