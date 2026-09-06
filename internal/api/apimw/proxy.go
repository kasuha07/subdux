package apimw

import (
	"fmt"
	"net"
	"strings"

	"github.com/labstack/echo/v4"
)

// ClientIPExtractor trusts only explicitly configured proxy CIDRs. With no
// proxies, forwarded headers never influence rate limiting or audit identity.
func ClientIPExtractor(cidrs string) (echo.IPExtractor, error) {
	if strings.TrimSpace(cidrs) == "" {
		return echo.ExtractIPDirect(), nil
	}
	opts := []echo.TrustOption{echo.TrustLoopback(false), echo.TrustLinkLocal(false), echo.TrustPrivateNet(false)}
	for _, raw := range strings.Split(cidrs, ",") {
		_, network, err := net.ParseCIDR(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("invalid TRUSTED_PROXY_CIDRS entry %q: %w", raw, err)
		}
		opts = append(opts, echo.TrustIPRange(network))
	}
	return echo.ExtractIPFromXFFHeader(opts...), nil
}
