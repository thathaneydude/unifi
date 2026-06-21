package cli_test

import (
	"context"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/internal/cli"
	"github.com/thathaneydude/unifi/unifi"
)

var _ = Describe("Request", func() {
	It("builds a request with path and query substitution (pure)", func() {
		conn := unifi.Local("host.example", "secret")
		op := cli.Operation{
			App:         unifi.AppNetwork,
			Method:      "GET",
			Path:        "/widgets/{id}",
			PathParams:  []cli.Param{{Name: "id", In: "path", Required: true}},
			QueryParams: []cli.Param{{Name: "limit", In: "query"}},
		}
		vals := cli.Values{
			Path:  map[string]string{"id": "w1"},
			Query: map[string]string{"limit": "5"},
		}

		req, err := cli.BuildRequest(context.Background(), conn, op, vals)
		Expect(err).NotTo(HaveOccurred())
		Expect(req.Method).To(Equal("GET"))
		Expect(req.URL.Path).To(Equal("/proxy/network/integration/widgets/w1"))
		Expect(req.URL.RawQuery).To(Equal("limit=5"))
	})

	It("executes against a server, applies auth, and returns body + status", func() {
		var gotPath, gotKey string
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotKey = r.Header.Get("X-API-KEY")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"w1"}]`))
		}))
		DeferCleanup(srv.Close)

		host := srv.Listener.Addr().String()
		conn := unifi.Local(host, "secret", unifi.WithHTTPClient(srv.Client()))

		op := cli.Operation{
			App:        unifi.AppNetwork,
			Method:     "GET",
			Path:       "/widgets/{id}",
			PathParams: []cli.Param{{Name: "id", In: "path", Required: true}},
		}
		vals := cli.Values{Path: map[string]string{"id": "w1"}}

		body, status, err := cli.Execute(context.Background(), conn, op, vals)
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(Equal(http.StatusOK))
		Expect(string(body)).To(Equal(`[{"id":"w1"}]`))
		Expect(gotPath).To(Equal("/proxy/network/integration/widgets/w1"))
		Expect(gotKey).To(Equal("secret"))
	})
})
