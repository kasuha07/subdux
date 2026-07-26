package backup

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/gorm"
)

var (
	ErrS3EndpointRequired    = serviceerr.New(serviceerr.KindInvalid, "s3_backup_endpoint_is_required", "s3 backup endpoint is required")
	ErrS3BucketRequired      = serviceerr.New(serviceerr.KindInvalid, "s3_backup_bucket_is_required", "s3 backup bucket is required")
	ErrS3CredentialsRequired = serviceerr.New(serviceerr.KindInvalid, "s3_backup_access_key_and_secret_are_required", "s3 backup access key id and secret access key are required")
	ErrS3EndpointInvalid     = serviceerr.New(serviceerr.KindInvalid, "s3_backup_endpoint_is_invalid", "s3 backup endpoint is invalid")
)

// s3BackupTimeout bounds a single S3 operation (upload, list page, delete).
const s3BackupTimeout = 60 * time.Second

// s3Target delivers backups to any S3-compatible object store (AWS S3, MinIO,
// Cloudflare R2, Backblaze B2, ...). The endpoint is admin-configured, so it is
// treated as administrator-trusted policy: the HTTP transport is proxy-aware
// but not subject to the user-facing SSRF filter, which lets an admin target a
// private-network MinIO. The archive it receives is already encrypted (when
// backup encryption is enabled), so the object store never sees plaintext.
type s3Target struct {
	client         *minio.Client
	bucket         string
	prefix         string
	retentionCount int
}

// s3Config is the decoded, validated form of a persisted s3 destination config.
type s3Config struct {
	endpointHost  string
	secure        bool
	region        string
	bucket        string
	prefix        string
	accessKeyID   string
	secretKey     string
	usePathStyle  bool
	skipTLSVerify bool
	retention     int
}

type s3DestinationConfig struct {
	Endpoint        string `json:"endpoint"`
	UseSSL          bool   `json:"use_ssl"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	Prefix          string `json:"prefix"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	UsePathStyle    bool   `json:"use_path_style"`
	SkipTLSVerify   bool   `json:"skip_tls_verify"`
	RetentionCount  int    `json:"retention_count"`
}

func newS3Target(config map[string]any, db *gorm.DB) (*s3Target, error) {
	parsed, err := parseS3Config(config)
	if err != nil {
		return nil, err
	}

	// The HTTP transport carries the system proxy configuration but intentionally
	// does not apply the SSRF filter, because backup endpoints are administrator
	// policy. See newBackupDestinationHTTPClient for the trust-boundary rationale
	// and for how skip_tls_verify is confined to this destination.
	httpClient, err := newBackupDestinationHTTPClient(db, s3BackupTimeout, parsed.skipTLSVerify)
	if err != nil {
		return nil, err
	}
	transport := httpClient.Transport

	lookup := minio.BucketLookupAuto
	if parsed.usePathStyle {
		lookup = minio.BucketLookupPath
	}

	client, err := minio.New(parsed.endpointHost, &minio.Options{
		Creds:        credentials.NewStaticV4(parsed.accessKeyID, parsed.secretKey, ""),
		Secure:       parsed.secure,
		Region:       parsed.region,
		Transport:    transport,
		BucketLookup: lookup,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrS3EndpointInvalid, err.Error())
	}

	return &s3Target{
		client:         client,
		bucket:         parsed.bucket,
		prefix:         parsed.prefix,
		retentionCount: parsed.retention,
	}, nil
}

func (t *s3Target) Type() string { return "s3" }

func (t *s3Target) RetentionCount() int { return t.retentionCount }

func (t *s3Target) Put(ctx context.Context, name string, r io.Reader, size int64) error {
	ctx, cancel := context.WithTimeout(ctx, s3BackupTimeout)
	defer cancel()

	_, err := t.client.PutObject(ctx, t.bucket, t.objectKey(name), r, size, minio.PutObjectOptions{
		ContentType: "application/zip",
	})
	return err
}

func (t *s3Target) List(ctx context.Context) ([]BackupObject, error) {
	ctx, cancel := context.WithTimeout(ctx, s3BackupTimeout)
	defer cancel()

	listPrefix := t.objectPrefix()
	objects := make([]BackupObject, 0)
	for object := range t.client.ListObjects(ctx, t.bucket, minio.ListObjectsOptions{
		Prefix: listPrefix,
		// Backups are stored directly below the configured prefix. A non-recursive
		// listing prevents nested objects from entering retention, and the
		// explicit direct-child check below keeps that invariant for providers
		// that still return nested Contents entries.
		Recursive: false,
	}) {
		if object.Err != nil {
			return nil, object.Err
		}
		name, ok := t.directObjectName(object.Key)
		if !ok || !backupFileNamePattern.MatchString(name) {
			continue
		}
		objects = append(objects, BackupObject{
			Name:       name,
			Size:       object.Size,
			ModifiedAt: object.LastModified,
			// Whether the archive is encrypted cannot be determined without
			// downloading it; the listing UI treats absent detection as false.
			Encrypted: false,
		})
	}
	return objects, nil
}

func (t *s3Target) Get(ctx context.Context, name string) (io.ReadCloser, int64, error) {
	if !isDirectS3ObjectName(name) || !backupFileNamePattern.MatchString(name) {
		return nil, 0, ErrInvalidBackupObjectName
	}
	// s3BackupTimeout already bounds a full archive upload in Put; a download of
	// the same archive is bounded the same way rather than introducing a second
	// knob. The cancel travels with the reader (see backupObjectReader) because
	// minio's object body is read after Get returns.
	ctx, cancel := context.WithTimeout(ctx, s3BackupTimeout)

	object, err := t.client.GetObject(ctx, t.bucket, t.objectKey(name), minio.GetObjectOptions{})
	if err != nil {
		cancel()
		return nil, 0, err
	}
	// GetObject is lazy; Stat is what actually issues the request, so a missing
	// object or a rejected credential surfaces here rather than mid-copy.
	info, err := object.Stat()
	if err != nil {
		_ = object.Close()
		cancel()
		return nil, 0, err
	}
	return &backupObjectReader{ReadCloser: object, cancel: cancel}, info.Size, nil
}

func (t *s3Target) Delete(ctx context.Context, name string) error {
	if !isDirectS3ObjectName(name) || !backupFileNamePattern.MatchString(name) {
		// Defensive: retention only ever passes back names List produced, which
		// are already direct, pattern-filtered children. Refuse anything else so
		// a bad caller can never remove an unrelated or nested object.
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, s3BackupTimeout)
	defer cancel()

	return t.client.RemoveObject(ctx, t.bucket, t.objectKey(name), minio.RemoveObjectOptions{})
}

// objectKey joins the configured prefix and the archive base name into the full
// object key, keeping delivery and retention symmetric.
func (t *s3Target) objectKey(name string) string {
	return t.objectPrefix() + name
}

func (t *s3Target) objectPrefix() string {
	if t.prefix == "" {
		return ""
	}
	return strings.TrimSuffix(t.prefix, "/") + "/"
}

func (t *s3Target) directObjectName(key string) (string, bool) {
	prefix := t.objectPrefix()
	if !strings.HasPrefix(key, prefix) {
		return "", false
	}
	name := strings.TrimPrefix(key, prefix)
	if !isDirectS3ObjectName(name) {
		return "", false
	}
	return name, true
}

func isDirectS3ObjectName(name string) bool {
	return name != "" && !strings.Contains(name, "/")
}

// parseS3Config validates and normalizes an s3 destination config. It performs
// no network I/O: the endpoint is only parsed for shape, so validation is safe
// to run at config-save time.
func parseS3Config(config map[string]any) (s3Config, error) {
	var raw s3DestinationConfig
	if err := decodeDestinationConfigStrict(config, "s3", &raw); err != nil {
		return s3Config{}, err
	}

	secure := true
	if _, present := config["use_ssl"]; present {
		secure = raw.UseSSL
	}
	retention, err := retentionCountFromConfig(config, raw.RetentionCount)
	if err != nil {
		return s3Config{}, err
	}

	cfg := s3Config{
		secure:        secure,
		region:        strings.TrimSpace(raw.Region),
		bucket:        strings.TrimSpace(raw.Bucket),
		prefix:        normalizeS3Prefix(raw.Prefix),
		accessKeyID:   strings.TrimSpace(raw.AccessKeyID),
		secretKey:     raw.SecretAccessKey,
		usePathStyle:  raw.UsePathStyle,
		skipTLSVerify: raw.SkipTLSVerify,
		retention:     retention,
	}

	endpointHost, secureFromScheme, hasScheme, err := normalizeS3Endpoint(raw.Endpoint)
	if err != nil {
		return s3Config{}, err
	}
	cfg.endpointHost = endpointHost
	if hasScheme {
		cfg.secure = secureFromScheme
	}

	if cfg.endpointHost == "" {
		return s3Config{}, ErrS3EndpointRequired
	}
	if cfg.bucket == "" {
		return s3Config{}, ErrS3BucketRequired
	}
	if cfg.accessKeyID == "" || strings.TrimSpace(cfg.secretKey) == "" {
		return s3Config{}, ErrS3CredentialsRequired
	}

	return cfg, nil
}

// normalizeS3Endpoint accepts either a bare host[:port] or a full URL without
// endpoint-specific path/query components. It returns the host[:port], whether
// TLS was implied by an explicit scheme, and whether a scheme was present at
// all. A bare host leaves the secure decision to the use_ssl field. A root path
// is accepted because it is equivalent to no path; all other URL components
// that MinIO would discard are rejected instead.
func normalizeS3Endpoint(raw string) (host string, secure bool, hasScheme bool, err error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false, false, nil
	}

	if strings.Contains(trimmed, "://") {
		parsed, parseErr := url.Parse(trimmed)
		if parseErr != nil || parsed.Host == "" || parsed.User != nil ||
			(parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" ||
			parsed.ForceQuery || parsed.Fragment != "" {
			return "", false, false, ErrS3EndpointInvalid
		}
		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "http" && scheme != "https" {
			return "", false, false, ErrS3EndpointInvalid
		}
		return parsed.Host, scheme == "https", true, nil
	}

	// Bare host[:port]; reject anything carrying URL components or userinfo.
	if strings.ContainsAny(trimmed, "/\\ ?#@") {
		return "", false, false, ErrS3EndpointInvalid
	}
	return trimmed, false, false, nil
}

// normalizeS3Prefix trims surrounding slashes and whitespace so objectKey can
// join it deterministically. An empty prefix means the bucket root.
func normalizeS3Prefix(raw string) string {
	trimmed := strings.Trim(strings.TrimSpace(raw), "/")
	return trimmed
}
