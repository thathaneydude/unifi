package unifi

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"time"
)

const defaultUserAgent = "unifi-go"

type config struct {
	httpClient *http.Client
	timeout    time.Duration
	userAgent  string
	tlsConfig  *tls.Config
	insecure   bool
	rootCAs    *x509.CertPool
}

// Option configures a Conn.
type Option func(*config)

// WithHTTPClient supplies a custom HTTP client. When a custom client is supplied,
// ALL TLS options (WithInsecureSkipVerify, WithRootCAs, WithTLSConfig) and
// WithTimeout are ignored entirely. Combining them with WithHTTPClient is a
// programming error.
func WithHTTPClient(c *http.Client) Option { return func(o *config) { o.httpClient = c } }

// WithTimeout sets the per-request timeout on the default client.
// If unset, a 30-second timeout is used.
func WithTimeout(d time.Duration) Option { return func(o *config) { o.timeout = d } }

// WithUserAgent overrides the User-Agent header.
func WithUserAgent(s string) Option { return func(o *config) { o.userAgent = s } }

// WithInsecureSkipVerify disables TLS verification. Local consoles only.
func WithInsecureSkipVerify() Option { return func(o *config) { o.insecure = true } }

// WithRootCAs trusts a specific CA pool (e.g. a pinned console certificate).
func WithRootCAs(p *x509.CertPool) Option { return func(o *config) { o.rootCAs = p } }

// WithTLSConfig provides a full TLS configuration.
func WithTLSConfig(t *tls.Config) Option { return func(o *config) { o.tlsConfig = t } }

func (o *config) resolveHTTPClient() *http.Client {
	if o.httpClient != nil {
		return o.httpClient
	}
	// Clone the caller's TLS config so we never mutate it. If none was
	// supplied, start from a fresh zero-value config.
	var tlsCfg *tls.Config
	if o.tlsConfig != nil {
		tlsCfg = o.tlsConfig.Clone()
	} else {
		tlsCfg = &tls.Config{} //nolint:gosec
	}
	if o.insecure {
		tlsCfg.InsecureSkipVerify = true //nolint:gosec
	}
	if o.rootCAs != nil {
		tlsCfg.RootCAs = o.rootCAs
	}
	timeout := o.timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{TLSClientConfig: tlsCfg},
	}
}
