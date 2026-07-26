package backup

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"strings"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"gorm.io/gorm"
)

// ErrBackupDestinationSecretRequiredForTest is returned when a connectivity
// probe would have to reuse a stored credential to reach an address the same
// request just changed.
//
// This is the one thing an ad-hoc probe must not do. Create/update/delete are
// gated by a step-up reauth ticket precisely because they decide where sensitive
// database backups (and the credentials that reach them) go; the probe carries
// no ticket because it only reads. Letting it pair a stored secret with a
// freshly typed endpoint would hand a hijacked admin session a way to post that
// secret to an attacker-controlled host without ever passing the step-up gate.
// Requiring the secret to be retyped keeps the probe's reach pinned to where the
// credential is already being sent.
var ErrBackupDestinationSecretRequiredForTest = serviceerr.New(
	serviceerr.KindInvalid,
	"backup_destination_secret_required_for_test",
	"re-enter the destination secret to test a changed endpoint",
)

// probeTransportSecretFields lists, per destination type, the secrets that are
// actually transmitted to the storage target. encryption_password is
// deliberately absent: it protects the archive bytes and never leaves the
// server, so a connectivity probe neither needs it nor should inherit it.
var probeTransportSecretFields = map[string][]string{
	"local":  {},
	"s3":     {"secret_access_key"},
	"webdav": {"password"},
}

// TestDestinationConfigInput is a connectivity probe against a config the admin
// is still editing, so nothing here has been persisted.
//
// DestinationID, when non-zero, names the saved destination the form was opened
// from. It is what allows secrets left blank (the server never returns them, so
// the form starts empty) to be inherited — but only under the address pinning
// described on ErrBackupDestinationSecretRequiredForTest. ClearedSecretFields
// mirrors the update path: a secret the admin explicitly cleared is probed as
// absent rather than revived from storage.
type TestDestinationConfigInput struct {
	Type                string
	Config              string
	DestinationID       uint
	ClearedSecretFields []string
}

// TestDestinationConfig runs the same read-only probe as TestDestination against
// an unsaved config, so an admin can validate a destination before committing to
// it. It returns the number of archives already present at the target.
func (s *Service) TestDestinationConfig(ctx context.Context, input TestDestinationConfigInput) (int, error) {
	destinationType := strings.ToLower(strings.TrimSpace(input.Type))

	var stored *model.BackupDestination
	if input.DestinationID != 0 {
		var destination model.BackupDestination
		if err := s.DB.First(&destination, input.DestinationID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return 0, ErrBackupDestinationNotFound
			}
			return 0, err
		}
		// The stored row is the authority on type: it is locked in the edit form
		// anyway, and taking it from the request would let a probe reinterpret
		// one type's stored secrets under another type's schema. It is normalized
		// like the request type because a row written by the backfill migration
		// (or by hand) could carry different casing, which would otherwise pass
		// isValidDestinationType and then silently miss both the secret-field and
		// target lookups, which key on the canonical form.
		destinationType = strings.ToLower(strings.TrimSpace(destination.Type))
		stored = &destination
	}
	if !isValidDestinationType(destinationType) {
		return 0, ErrInvalidBackupDestinationType
	}

	config, err := parseDestinationConfigMap(input.Config)
	if err != nil {
		return 0, err
	}
	// A probe exercises reachability only, so the fields that describe the plan
	// rather than the connection are dropped rather than validated. An admin
	// checking whether an endpoint answers should not first have to satisfy when
	// the plan fires, how its archive is encrypted, or how many archives it
	// keeps.
	config = destinationConnectivityConfig(config)

	if destinationType == "local" {
		if rawDir, ok := config["dir"].(string); ok {
			normalized, err := normalizeBackupLocalDir(rawDir)
			if err != nil {
				return 0, err
			}
			config["dir"] = normalized
		}
	}

	if stored != nil {
		if err := inheritProbeSecrets(destinationType, *stored, config, input.ClearedSecretFields); err != nil {
			return 0, err
		}
	}

	target, err := newBackupTargetFromConfig(destinationType, config, s.DB)
	if err != nil {
		return 0, err
	}
	objects, err := target.List(ctx)
	if err != nil {
		return 0, err
	}
	return len(objects), nil
}

// destinationConnectivityConfig copies config down to the fields that decide
// whether the storage target can be reached.
//
// retention_count is dropped alongside the schedule even though it is not a
// schedule field: List never consults it, but retentionCountFromConfig rejects
// anything outside 1..1000, which would fail a probe over a number that has no
// bearing on reachability. Its absence is well defined — retentionCountFromConfig
// returns the default when the key is missing.
func destinationConnectivityConfig(config map[string]any) map[string]any {
	connectivity := make(map[string]any, len(config))
	for field, raw := range config {
		if _, isSchedule := scheduleConfigFields[field]; isSchedule {
			continue
		}
		if field == "retention_count" {
			continue
		}
		connectivity[field] = raw
	}
	return connectivity
}

// inheritProbeSecrets fills in transport secrets the admin left blank from the
// saved destination, but only while the probe still points at the address that
// credential is already delivered to. Anything else is refused rather than
// silently probed with the wrong credential, so the admin learns why instead of
// reading an unexplained authentication failure.
func inheritProbeSecrets(
	destinationType string,
	stored model.BackupDestination,
	config map[string]any,
	clearedFields []string,
) error {
	blank := make([]string, 0, len(probeTransportSecretFields[destinationType]))
	for _, field := range probeTransportSecretFields[destinationType] {
		if slices.Contains(clearedFields, field) {
			// The admin asked for the stored secret to go away, so the probe
			// tests exactly that: anonymous access, not the old credential.
			config[field] = ""
			continue
		}
		if value, ok := config[field].(string); ok && strings.TrimSpace(value) != "" {
			continue
		}
		blank = append(blank, field)
	}
	if len(blank) == 0 {
		return nil
	}

	plain, err := decryptDestinationConfig(stored.Config)
	if err != nil {
		return ErrInvalidBackupDestinationConfig
	}
	storedConfig, err := parseDestinationConfigMap(plain)
	if err != nil {
		return err
	}

	// Resolve the probe's address first: a config the admin has broken in some
	// other way (blanked bucket, malformed URL) should report that error rather
	// than being reported as a missing secret.
	probeAddress, err := probeCredentialAddress(destinationType, config)
	if err != nil {
		return err
	}
	storedAddress, storedErr := probeCredentialAddress(destinationType, storedConfig)
	pinned := storedErr == nil && credentialAddressPinned(storedAddress, probeAddress)

	for _, field := range blank {
		storedValue, ok := storedConfig[field].(string)
		if !ok || strings.TrimSpace(storedValue) == "" {
			// Nothing stored to inherit. Leave the field blank and let the
			// target constructor rule on it, which keeps the per-type answer in
			// one place: anonymous WebDAV is legal, S3 without a secret is not.
			continue
		}
		if !pinned {
			return ErrBackupDestinationSecretRequiredForTest
		}
		config[field] = storedValue
	}
	return nil
}

// credentialAddress identifies where a config would transmit its credential and
// how well protected it is on the way there.
type credentialAddress struct {
	kind string
	host string
	// account is the identity the credential authenticates as, where the protocol
	// sends one. It is part of the address because a credential offered under a
	// different account is a different disclosure: see the WebDAV case below.
	account       string
	secure        bool
	skipTLSVerify bool
}

// credentialAddressPinned reports whether probing under probe would keep a
// stored credential at least as well protected as stored already keeps it.
//
// The transport posture is compared as an ordering rather than for equality, and
// that asymmetry is the point. Equality would also refuse the safe direction: an
// admin who switches skip_tls_verify off, or who moves an endpoint from http to
// https, has strictly reduced the credential's exposure. Refusing those is not
// just needless friction — it reports "re-enter the destination secret to test a
// changed endpoint" when no endpoint changed, which reads as a bug.
//
// The host is still compared exactly, because that is the part an attacker would
// have to move to receive anything.
func credentialAddressPinned(stored, probe credentialAddress) bool {
	if stored.kind != probe.kind || stored.host != probe.host {
		return false
	}
	if stored.account != probe.account {
		return false
	}
	// Relaxing certificate verification widens who can answer for the host.
	if probe.skipTLSVerify && !stored.skipTLSVerify {
		return false
	}
	// Dropping TLS puts a credential that was protected onto the clear.
	if !probe.secure && stored.secure {
		return false
	}
	return true
}

// probeCredentialAddress resolves where this config would transmit its
// credential, and under what transport protection.
//
// It resolves through the same parsers the runtime uses so that equivalent
// spellings compare equal ("s3.example.com" and "https://s3.example.com" are one
// address) and so this pinning can never drift from how the target is actually
// built. skip_tls_verify is part of the identity because relaxing verification is
// another way to change who can receive a credential.
//
// Bucket, prefix, region, and the WebDAV subfolder are deliberately excluded, so
// that ordinary edits do not force a needless secret retype. For WebDAV that is
// simply because they cannot move the credential: Basic auth transmits the
// password, but only ever to the origin and account pinned above.
//
// S3 needs the sharper argument, because those fields are NOT purely path
// components. Under minio-go's BucketLookupAuto, virtual-host addressing sends
// requests to <bucket>.<endpoint>, and for Amazon/Google endpoints the region
// selects a regional host, so bucket, region, and use_path_style can each change
// the host actually contacted.
//
// That is safe here for a reason specific to S3 rather than by accident: newS3
// Target signs with SigV4 (credentials.NewStaticV4), which derives a per-request
// signature and never puts the secret access key on the wire. A redirected probe
// would disclose only a signature scoped to one canonical request, plus the
// access key id — not a secret, shown in the form in plaintext, and pinned above
// anyway. The rewrite also cannot leave the provider's own domain, because
// CheckValidBucketName admits no '/' or '@' in a bucket name. Moving the
// credential's address for real still requires the endpoint, which is pinned.
//
// Do not "fix" this comment by asserting these fields keep the same host. They
// do not, and the argument above is what makes their exclusion sound.
func probeCredentialAddress(destinationType string, config map[string]any) (credentialAddress, error) {
	// The parsers reject a blank credential before they resolve an endpoint, and
	// a blank credential is exactly the case being adjudicated here, so the
	// address is resolved against a placeholder. The placeholder is never used
	// to reach anything: only the returned identity string is.
	resolvable := make(map[string]any, len(config)+1)
	for field, raw := range config {
		resolvable[field] = raw
	}
	for _, field := range probeTransportSecretFields[destinationType] {
		if value, ok := resolvable[field].(string); !ok || strings.TrimSpace(value) == "" {
			resolvable[field] = "placeholder"
		}
	}

	switch destinationType {
	case "s3":
		parsed, err := parseS3Config(resolvable)
		if err != nil {
			return credentialAddress{}, err
		}
		return credentialAddress{
			kind: "s3",
			host: parsed.endpointHost,
			// Pinning the access key id costs nothing real: an old secret paired
			// with a new key id could only ever fail the signature, so refusing
			// with "re-enter the secret" beats inheriting into a guaranteed
			// authentication error.
			account:       parsed.accessKeyID,
			secure:        parsed.secure,
			skipTLSVerify: parsed.skipTLSVerify,
		}, nil
	case "webdav":
		parsed, err := parseWebDAVConfig(resolvable)
		if err != nil {
			return credentialAddress{}, err
		}
		base, err := url.Parse(parsed.baseURL)
		if err != nil {
			return credentialAddress{}, ErrWebDAVURLInvalid
		}
		return credentialAddress{
			kind: "webdav",
			host: base.Host,
			// username is pinned even though it is not itself a secret. Leaving it
			// free would let a blank password be inherited and offered under a
			// different account on the same server — a password-testing oracle
			// against the victim's own WebDAV host, which is how a reused backup
			// password gets discovered.
			account:       parsed.username,
			secure:        base.Scheme == "https",
			skipTLSVerify: parsed.skipTLSVerify,
		}, nil
	default:
		// local has no transmitted credential, so every config shares one
		// trivially-pinned address.
		return credentialAddress{kind: destinationType}, nil
	}
}
