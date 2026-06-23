package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/unifi"
)

var _ = Describe("selectSite", func() {
	sites := []site{
		{ID: "id-default", Name: "Default", InternalReference: "default"},
		{ID: "id-lab", Name: "Lab", InternalReference: "lab123"},
	}

	It("auto-selects the only site when none requested", func() {
		id, err := selectSite(sites[:1], "")
		Expect(err).NotTo(HaveOccurred())
		Expect(id).To(Equal("id-default"))
	})

	It("errors when multiple sites and none requested", func() {
		_, err := selectSite(sites, "")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("multiple sites"))
	})

	It("matches by internalReference (default), name, and id, case-insensitively", func() {
		id, err := selectSite(sites, "default")
		Expect(err).NotTo(HaveOccurred())
		Expect(id).To(Equal("id-default"))

		id, err = selectSite(sites, "lab")
		Expect(err).NotTo(HaveOccurred())
		Expect(id).To(Equal("id-lab"))

		id, err = selectSite(sites, "ID-LAB")
		Expect(err).NotTo(HaveOccurred())
		Expect(id).To(Equal("id-lab"))
	})

	It("errors when nothing matches", func() {
		_, err := selectSite(sites, "nope")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no site matches"))
	})
})

var _ = Describe("parseSites", func() {
	It("reads the data envelope", func() {
		sites, err := parseSites([]byte(`{"data":[{"id":"a","name":"A","internalReference":"default"}]}`))
		Expect(err).NotTo(HaveOccurred())
		Expect(sites).To(HaveLen(1))
		Expect(sites[0]).To(Equal(site{ID: "a", Name: "A", InternalReference: "default"}))
	})
})

var _ = Describe("siteId injection", func() {
	var (
		server   *httptest.Server
		gotPath  string
		out      *bytes.Buffer
		deps     runDeps
		op       Operation
		resolved bool
	)

	BeforeEach(func() {
		gotPath = ""
		resolved = false
		out = &bytes.Buffer{}
		server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
		}))
		DeferCleanup(server.Close)

		conn := unifi.Local(server.Listener.Addr().String(), "secret", unifi.WithHTTPClient(server.Client()))
		deps = runDeps{
			connFn: func(unifi.App) (*unifi.Conn, error) { return conn, nil },
			format: func() Format { return FormatJSON },
			resolveSite: func(context.Context, *unifi.Conn, string) (string, error) {
				resolved = true
				return "RID", nil
			},
			site:   func() string { return "" },
			stdout: out,
		}
		op = Operation{
			App: unifi.AppNetwork, ID: "GetNetworks", Method: http.MethodGet,
			Path:       "/v1/sites/{siteId}/networks",
			PathParams: []Param{{Name: "siteId", In: "path", Required: true}},
		}
	})

	run := func(args ...string) error {
		cmd := NewAppCommand(unifi.AppNetwork, []Operation{op}, deps)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs(args)
		return cmd.Execute()
	}

	It("injects the resolved siteId when --siteId is not set", func() {
		Expect(run("GetNetworks")).To(Succeed())
		Expect(resolved).To(BeTrue())
		Expect(gotPath).To(HaveSuffix("/v1/sites/RID/networks"))
	})

	It("uses an explicit --siteId and does not resolve", func() {
		Expect(run("GetNetworks", "--siteId", "explicit")).To(Succeed())
		Expect(resolved).To(BeFalse())
		Expect(gotPath).To(HaveSuffix("/v1/sites/explicit/networks"))
	})
})
