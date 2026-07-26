package backup

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/service/outbound"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"gorm.io/gorm"
)

var (
	ErrWebDAVURLRequired = serviceerr.New(serviceerr.KindInvalid, "webdav_backup_url_is_required", "webdav backup url is required")
	ErrWebDAVURLInvalid  = serviceerr.New(serviceerr.KindInvalid, "webdav_backup_url_is_invalid", "webdav backup url is invalid")
)

const webdavTimeout = 60 * time.Second

// webdavTarget delivers backups to a WebDAV server (Nextcloud, ownCloud, Apache
// mod_dav, ...). Like the S3 target, the endpoint is admin-configured and thus
// administrator-trusted: the HTTP client is proxy-aware but not subject to the
// user-facing SSRF filter, and the archive it uploads is already encrypted when
// backup encryption is enabled.
type webdavTarget struct {
	client         *http.Client
	baseURL        string // collection URL the archives live under, no trailing slash
	username       string
	password       string
	retentionCount int
}

type webdavConfig struct {
	baseURL       string
	username      string
	password      string
	skipTLSVerify bool
	retention     int
}

type webdavDestinationConfig struct {
	URL            string `json:"url"`
	Path           string `json:"path"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	SkipTLSVerify  bool   `json:"skip_tls_verify"`
	RetentionCount int    `json:"retention_count"`
}

func newWebDAVTarget(config map[string]any, db *gorm.DB) (*webdavTarget, error) {
	parsed, err := parseWebDAVConfig(config)
	if err != nil {
		return nil, err
	}

	// See newBackupDestinationHTTPClient for the trust-boundary rationale and
	// for how skip_tls_verify stays confined to this destination.
	client, err := newBackupDestinationHTTPClient(db, webdavTimeout, parsed.skipTLSVerify)
	if err != nil {
		return nil, err
	}

	return &webdavTarget{
		client:         client,
		baseURL:        parsed.baseURL,
		username:       parsed.username,
		password:       parsed.password,
		retentionCount: parsed.retention,
	}, nil
}

func (t *webdavTarget) Type() string { return "webdav" }

func (t *webdavTarget) RetentionCount() int { return t.retentionCount }

func (t *webdavTarget) Put(ctx context.Context, name string, r io.Reader, size int64) error {
	ctx, cancel := context.WithTimeout(ctx, webdavTimeout)
	defer cancel()

	if !isSafeBackupFileName(name) {
		return ErrInvalidBackupObjectName
	}
	if err := t.ensureCollection(ctx); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, t.objectURL(name), r)
	if err != nil {
		return err
	}
	t.authorize(req)
	req.ContentLength = size
	req.Header.Set("Content-Type", "application/zip")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer drainAndClose(resp.Body)

	if !isWebDAVSuccess(resp.StatusCode) {
		return webdavStatusError("upload", resp.StatusCode)
	}
	return nil
}

func (t *webdavTarget) List(ctx context.Context) ([]BackupObject, error) {
	ctx, cancel := context.WithTimeout(ctx, webdavTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "PROPFIND", t.baseURL+"/", nil)
	if err != nil {
		return nil, err
	}
	t.authorize(req)
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer drainAndClose(resp.Body)

	// 404 on the collection means nothing has been uploaded yet: an empty list,
	// not an error, so a first run's retention pass is a no-op.
	if resp.StatusCode == http.StatusNotFound {
		return []BackupObject{}, nil
	}
	if resp.StatusCode != http.StatusMultiStatus && !isWebDAVSuccess(resp.StatusCode) {
		return nil, webdavStatusError("list", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}

	var multistatus webdavMultistatus
	if err := xml.Unmarshal(body, &multistatus); err != nil {
		return nil, invalidWebDAVResponse(err)
	}

	objects := make([]BackupObject, 0, len(multistatus.Responses))
	for _, response := range multistatus.Responses {
		if response.isCollection() {
			continue
		}
		name, ok := t.directHrefName(response.Href)
		if !ok {
			continue
		}
		modifiedAt, err := response.lastModified()
		if err != nil {
			// Retention orders by this value. Refuse the whole listing when a
			// candidate archive cannot be ordered instead of returning the zero
			// time and allowing retention to delete an arbitrary archive.
			return nil, invalidWebDAVResponse(err)
		}
		objects = append(objects, BackupObject{
			Name:       name,
			Size:       response.contentLength(),
			ModifiedAt: modifiedAt,
			// WebDAV does not expose archive-internal encryption; the listing UI
			// treats absent detection as false.
			Encrypted: false,
		})
	}
	return objects, nil
}

func (t *webdavTarget) Delete(ctx context.Context, name string) error {
	if !isSafeBackupFileName(name) {
		// Defensive: retention only passes back names List produced. Refuse
		// anything else so a bad caller can never delete an unrelated resource.
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, webdavTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, t.objectURL(name), nil)
	if err != nil {
		return err
	}
	t.authorize(req)

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer drainAndClose(resp.Body)

	// Deleting something already gone is not an error.
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if !isWebDAVSuccess(resp.StatusCode) {
		return webdavStatusError("delete", resp.StatusCode)
	}
	return nil
}

// ensureCollection creates the configured collection. 405 means the collection
// already exists on common WebDAV servers; 301 is retained for servers that
// report their canonical collection URL as a compatibility response. Other
// statuses and transport failures must reach the caller.
func (t *webdavTarget) ensureCollection(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "MKCOL", t.baseURL+"/", nil)
	if err != nil {
		return err
	}
	t.authorize(req)
	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	drainAndClose(resp.Body)
	if isWebDAVSuccess(resp.StatusCode) || resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusMovedPermanently {
		return nil
	}
	return webdavStatusError("create collection", resp.StatusCode)
}

func (t *webdavTarget) authorize(req *http.Request) {
	if t.username != "" || t.password != "" {
		req.SetBasicAuth(t.username, t.password)
	}
}

func (t *webdavTarget) objectURL(name string) string {
	return t.baseURL + "/" + url.PathEscape(name)
}

// parseWebDAVConfig validates and normalizes a webdav destination config. It
// performs no network I/O, so it is safe to run at config-save time. The base
// collection URL is the join of the configured url and optional path, with a
// trailing slash trimmed for deterministic joining.
func parseWebDAVConfig(config map[string]any) (webdavConfig, error) {
	var raw webdavDestinationConfig
	if err := decodeDestinationConfigStrict(config, "webdav", &raw); err != nil {
		return webdavConfig{}, err
	}

	retention, err := retentionCountFromConfig(config, raw.RetentionCount)
	if err != nil {
		return webdavConfig{}, err
	}

	rawURL := strings.TrimSpace(raw.URL)
	if rawURL == "" {
		return webdavConfig{}, ErrWebDAVURLRequired
	}

	parsedURL, err := outbound.ValidateHTTPURL(rawURL, "webdav backup url", false)
	if err != nil {
		return webdavConfig{}, fmt.Errorf("%w: %s", ErrWebDAVURLInvalid, err.Error())
	}
	if parsedURL.User != nil {
		return webdavConfig{}, fmt.Errorf("%w: URL must not contain embedded credentials", ErrWebDAVURLInvalid)
	}
	if parsedURL.RawQuery != "" || parsedURL.ForceQuery || parsedURL.Fragment != "" || hasWebDAVDotSegment(parsedURL.Path) {
		return webdavConfig{}, fmt.Errorf("%w: URL contains an unsafe path or delimiter", ErrWebDAVURLInvalid)
	}

	extra, err := normalizeWebDAVPath(raw.Path)
	if err != nil {
		return webdavConfig{}, fmt.Errorf("%w: %s", ErrWebDAVURLInvalid, err.Error())
	}
	parsedURL.Path = strings.TrimRight(parsedURL.Path, "/")
	parsedURL.RawPath = ""
	if extra != "" {
		parsedURL.Path += "/" + extra
	}
	base := strings.TrimRight(parsedURL.String(), "/")

	return webdavConfig{
		baseURL:       base,
		username:      strings.TrimSpace(raw.Username),
		password:      raw.Password,
		skipTLSVerify: raw.SkipTLSVerify,
		retention:     retention,
	}, nil
}

func normalizeWebDAVPath(raw string) (string, error) {
	trimmed := strings.Trim(strings.TrimSpace(raw), "/")
	if trimmed == "" {
		return "", nil
	}
	if strings.ContainsAny(trimmed, `\\?#`) {
		return "", fmt.Errorf("path contains an unsafe delimiter")
	}
	parts := strings.Split(trimmed, "/")
	for i, part := range parts {
		decoded, err := url.PathUnescape(part)
		if err != nil || decoded == "." || decoded == ".." || strings.ContainsAny(decoded, `/\\`) {
			return "", fmt.Errorf("path contains an unsafe segment")
		}
		parts[i] = decoded
	}
	return strings.Join(parts, "/"), nil
}

func hasWebDAVDotSegment(rawPath string) bool {
	decoded, err := url.PathUnescape(rawPath)
	if err != nil {
		return true
	}
	for _, segment := range strings.Split(decoded, "/") {
		if segment == "." || segment == ".." {
			return true
		}
	}
	return false
}

func isWebDAVSuccess(status int) bool {
	return status >= 200 && status < 300
}

func webdavStatusError(op string, status int) error {
	return serviceerr.New(serviceerr.KindInternal, "webdav_backup_request_failed", fmt.Sprintf("webdav %s failed with status %d", op, status))
}

func invalidWebDAVResponse(err error) error {
	return serviceerr.New(serviceerr.KindInternal, "webdav_backup_response_invalid", fmt.Sprintf("invalid webdav response: %s", err.Error()))
}

func drainAndClose(body io.ReadCloser) {
	if body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(body, 1<<20))
	_ = body.Close()
}

func (t *webdavTarget) directHrefName(href string) (string, bool) {
	base, err := url.Parse(t.baseURL)
	if err != nil {
		return "", false
	}
	candidate, err := url.Parse(strings.TrimSpace(href))
	if err != nil || candidate.RawQuery != "" || candidate.ForceQuery || candidate.Fragment != "" || candidate.User != nil {
		return "", false
	}
	if candidate.IsAbs() && (candidate.Scheme != base.Scheme || candidate.Host != base.Host) {
		return "", false
	}
	if candidate.Host != "" && candidate.Host != base.Host {
		return "", false
	}
	if hasWebDAVDotSegment(candidate.Path) {
		return "", false
	}
	collectionPath := strings.TrimRight(base.Path, "/")
	candidatePath := strings.TrimRight(candidate.Path, "/")
	prefix := collectionPath + "/"
	if !strings.HasPrefix(candidatePath, prefix) {
		return "", false
	}
	relative := strings.TrimPrefix(candidatePath, prefix)
	if relative == "" || strings.Contains(relative, "/") {
		return "", false
	}
	name, err := url.PathUnescape(relative)
	if err != nil || !isSafeBackupFileName(name) {
		return "", false
	}
	return name, true
}

// WebDAV multistatus XML. Tags are bound to the standard "DAV:" namespace URI so
// parsing is agnostic to the prefix a given server uses (d:, D:, ...).
type webdavMultistatus struct {
	XMLName   xml.Name         `xml:"DAV: multistatus"`
	Responses []webdavResponse `xml:"DAV: response"`
}

type webdavResponse struct {
	Href     string           `xml:"DAV: href"`
	Propstat []webdavPropstat `xml:"DAV: propstat"`
}

type webdavPropstat struct {
	Prop webdavProp `xml:"DAV: prop"`
}

type webdavProp struct {
	ContentLength int64              `xml:"DAV: getcontentlength"`
	LastModified  string             `xml:"DAV: getlastmodified"`
	ResourceType  webdavResourceType `xml:"DAV: resourcetype"`
}

type webdavResourceType struct {
	Collection *struct{} `xml:"DAV: collection"`
}

func (r webdavResponse) isCollection() bool {
	for _, propstat := range r.Propstat {
		if propstat.Prop.ResourceType.Collection != nil {
			return true
		}
	}
	return false
}

func (r webdavResponse) contentLength() int64 {
	for _, propstat := range r.Propstat {
		if propstat.Prop.ContentLength > 0 {
			return propstat.Prop.ContentLength
		}
	}
	return 0
}

func (r webdavResponse) lastModified() (time.Time, error) {
	var parseErr error
	for _, propstat := range r.Propstat {
		raw := strings.TrimSpace(propstat.Prop.LastModified)
		if raw == "" {
			continue
		}
		if parsed, err := http.ParseTime(raw); err == nil {
			return parsed, nil
		} else {
			parseErr = err
		}
	}
	if parseErr != nil {
		return time.Time{}, fmt.Errorf("invalid getlastmodified: %w", parseErr)
	}
	return time.Time{}, fmt.Errorf("getlastmodified is missing")
}
