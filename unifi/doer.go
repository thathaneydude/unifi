package unifi

import "net/http"

// HTTPDoer is the minimal HTTP interface the SDK depends on; *http.Client
// satisfies it. It is the seam used to drive generated clients in tests
// without real network access.
//
//counterfeiter:generate . HTTPDoer
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}
