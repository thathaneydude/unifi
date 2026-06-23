package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/unifi"
)

var _ = Describe("selectConsole", func() {
	consoles := []console{
		{ID: "id-home", Name: "Home", Model: "UniFi Dream Machine SE", Shortname: "UDMPROSE"},
		{ID: "id-nvr", Name: "Network Video Recorder", Model: "UniFi Network Video Recorder", Shortname: "UNVR"},
	}

	It("auto-selects the only console when none requested", func() {
		id, err := selectConsole(consoles[:1], "")
		Expect(err).NotTo(HaveOccurred())
		Expect(id).To(Equal("id-home"))
	})

	It("errors when multiple consoles and none requested", func() {
		_, err := selectConsole(consoles, "")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("multiple consoles"))
	})

	It("matches by name, shortname, model, and id, case-insensitively", func() {
		id, err := selectConsole(consoles, "home")
		Expect(err).NotTo(HaveOccurred())
		Expect(id).To(Equal("id-home"))

		id, err = selectConsole(consoles, "UDMPROSE")
		Expect(err).NotTo(HaveOccurred())
		Expect(id).To(Equal("id-home"))

		id, err = selectConsole(consoles, "unifi network video recorder")
		Expect(err).NotTo(HaveOccurred())
		Expect(id).To(Equal("id-nvr"))

		id, err = selectConsole(consoles, "ID-NVR")
		Expect(err).NotTo(HaveOccurred())
		Expect(id).To(Equal("id-nvr"))
	})

	It("errors when nothing matches, listing the consoles", func() {
		_, err := selectConsole(consoles, "nope")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no console matches"))
		Expect(err.Error()).To(ContainSubstring("Home (UDMPROSE)"))
	})
})

var _ = Describe("parseHostsPage", func() {
	It("reads id, name (falling back to hostname), hardware, ip, owner and nextToken", func() {
		body := []byte(`{"data":[
			{"id":"id-1","ipAddress":"10.0.0.1","owner":true,
			 "reportedState":{"name":"Home","hostname":"home-host",
			   "hardware":{"name":"UniFi Dream Machine SE","shortname":"UDMPROSE"}}},
			{"id":"id-2","ipAddress":"10.0.0.2","owner":false,
			 "reportedState":{"hostname":"nvr-host",
			   "hardware":{"name":"UniFi Network Video Recorder","shortname":"UNVR"}}}
		],"nextToken":"tok"}`)
		got, next, err := parseHostsPage(body)
		Expect(err).NotTo(HaveOccurred())
		Expect(next).To(Equal("tok"))
		Expect(got).To(HaveLen(2))
		Expect(got[0]).To(Equal(console{ID: "id-1", Name: "Home", Model: "UniFi Dream Machine SE", Shortname: "UDMPROSE", IP: "10.0.0.1", Owner: true}))
		// name empty -> falls back to hostname.
		Expect(got[1].Name).To(Equal("nvr-host"))
	})

	It("returns empty nextToken when absent", func() {
		_, next, err := parseHostsPage([]byte(`{"data":[]}`))
		Expect(err).NotTo(HaveOccurred())
		Expect(next).To(BeEmpty())
	})
})

var _ = Describe("consoles list command", func() {
	var (
		server   *httptest.Server
		out      *bytes.Buffer
		gotPaths []string
	)

	BeforeEach(func() {
		out = &bytes.Buffer{}
		gotPaths = nil
		// Two pages exercise nextToken pagination.
		server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPaths = append(gotPaths, r.URL.Path)
			w.WriteHeader(http.StatusOK)
			if r.URL.Query().Get("nextToken") == "" {
				_, _ = w.Write([]byte(`{"data":[{"id":"id-1","ipAddress":"10.0.0.1","owner":true,"reportedState":{"name":"Home","hardware":{"name":"UDM SE","shortname":"UDMPROSE"}}}],"nextToken":"p2"}`))
				return
			}
			_, _ = w.Write([]byte(`{"data":[{"id":"id-2","ipAddress":"10.0.0.2","owner":false,"reportedState":{"name":"NVR","hardware":{"name":"NVR","shortname":"UNVR"}}}]}`))
		}))
		DeferCleanup(server.Close)
	})

	// accountConn points the synthetic /v1/hosts op at the test server by using a
	// Local-style base; BuildRequest joins base + op.Path so the path is /v1/hosts.
	accountConn := func(srv *httptest.Server) func() (*unifi.Conn, error) {
		return func() (*unifi.Conn, error) {
			return unifi.Local(srv.Listener.Addr().String(), "secret", unifi.WithHTTPClient(srv.Client())), nil
		}
	}

	run := func(format Format, args ...string) error {
		cmd := newConsolesCommand(out, accountConn(server), func() Format { return format }, func() RenderOptions { return RenderOptions{} })
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs(args)
		return cmd.Execute()
	}

	It("paginates and emits a compact console array", func() {
		Expect(run(FormatJSON, "list")).To(Succeed())
		Expect(gotPaths).To(HaveLen(2)) // followed nextToken once
		var views []consoleView
		Expect(json.Unmarshal(out.Bytes(), &views)).To(Succeed())
		Expect(views).To(HaveLen(2))
		Expect(views[0].ID).To(Equal("id-1"))
		Expect(views[0].Name).To(Equal("Home"))
		Expect(views[1].ID).To(Equal("id-2"))
	})

	It("projects fields when --fields is applied via render options", func() {
		cmd := newConsolesCommand(out, accountConn(server), func() Format { return FormatJSON }, func() RenderOptions { return RenderOptions{Fields: []string{"id"}} })
		cmd.SetArgs([]string{"list"})
		Expect(cmd.Execute()).To(Succeed())
		var views []map[string]any
		Expect(json.Unmarshal(out.Bytes(), &views)).To(Succeed())
		Expect(views[0]).To(HaveKey("id"))
		Expect(views[0]).NotTo(HaveKey("name"))
	})

	It("errors on an unknown subcommand", func() {
		err := run(FormatJSON, "bogus")
		Expect(err).To(HaveOccurred())
	})
})
