package cli

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/unifi"
)

func loadFixtureInternal() *openapi3.T {
	data, err := os.ReadFile("testdata/fixture.openapi.json")
	Expect(err).NotTo(HaveOccurred())
	doc, err := openapi3.NewLoader().LoadFromData(data)
	Expect(err).NotTo(HaveOccurred())
	return doc
}

var _ = Describe("runOperation safety gate", func() {
	var (
		server   *httptest.Server
		called   bool
		out      *bytes.Buffer
		createOp Operation
		deps     runDeps
	)

	BeforeEach(func() {
		called = false
		out = &bytes.Buffer{}
		server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"ok":true}`))
		}))
		DeferCleanup(server.Close)

		host := server.Listener.Addr().String()
		conn := unifi.Local(host, "secret", unifi.WithHTTPClient(server.Client()))
		deps = runDeps{
			connFn: func(unifi.App) (*unifi.Conn, error) { return conn, nil },
			format: func() Format { return FormatJSON },
			stdout: out,
		}

		for _, op := range OperationsFor(loadFixtureInternal(), unifi.AppNetwork, "v1.0.0") {
			if op.ID == "CreateWidget" {
				createOp = op
			}
		}
	})

	run := func(args ...string) error {
		cmd := NewAppCommand(unifi.AppNetwork, []Operation{createOp}, deps)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs(args)
		return cmd.Execute()
	}

	It("rejects a mutating op without --confirm and does not call the server", func() {
		err := run("CreateWidget", "--body", `{"x":1}`)
		Expect(err).To(HaveOccurred())
		var cerr *CLIError
		Expect(errors.As(err, &cerr)).To(BeTrue())
		Expect(cerr.ExitCode()).To(Equal(1))
		Expect(called).To(BeFalse())
	})

	It("prints a dry-run preview without calling the server", func() {
		err := run("CreateWidget", "--body", `{"x":1}`, "--dry-run")
		Expect(err).NotTo(HaveOccurred())
		Expect(called).To(BeFalse())
		Expect(out.String()).To(ContainSubstring(`"dryRun"`))
	})

	It("executes when --confirm is provided", func() {
		err := run("CreateWidget", "--body", `{"x":1}`, "--confirm")
		Expect(err).NotTo(HaveOccurred())
		Expect(called).To(BeTrue())
		Expect(out.String()).To(ContainSubstring("ok"))
	})
})

var _ = Describe("operation flag handling", func() {
	var (
		putItem Operation
		out     *bytes.Buffer
		deps    runDeps
	)

	BeforeEach(func() {
		out = &bytes.Buffer{}
		conn := unifi.Local("127.0.0.1", "secret")
		deps = runDeps{
			connFn: func(unifi.App) (*unifi.Conn, error) { return conn, nil },
			format: func() Format { return FormatJSON },
			stdout: out,
		}
		data, err := os.ReadFile("testdata/edgecases.openapi.json")
		Expect(err).NotTo(HaveOccurred())
		doc, err := openapi3.NewLoader().LoadFromData(data)
		Expect(err).NotTo(HaveOccurred())
		for _, op := range OperationsFor(doc, unifi.AppNetwork, "v1.0.0") {
			if op.ID == "PutItem" {
				putItem = op
			}
		}
	})

	run := func(args ...string) error {
		cmd := NewAppCommand(unifi.AppNetwork, []Operation{putItem}, deps)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs(args)
		return cmd.Execute()
	}

	It("builds the command tree without panicking on a path/query name collision", func() {
		Expect(func() { NewAppCommand(unifi.AppNetwork, []Operation{putItem}, deps) }).NotTo(Panic())
	})

	It("routes --id to the path and --query-id to the query string", func() {
		err := run("PutItem", "--id", "abc", "--query-id", "xyz", "--body", "{}", "--dry-run")
		Expect(err).NotTo(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("/v1/items/abc"))
		Expect(out.String()).To(ContainSubstring("id=xyz"))
	})

	It("sends a deliberately empty required path value instead of dropping it", func() {
		err := run("PutItem", "--id", "", "--query-id", "xyz", "--body", "{}", "--dry-run")
		Expect(err).NotTo(HaveOccurred())
		Expect(out.String()).NotTo(ContainSubstring("{id}"))
	})
})

var _ = Describe("BuildRequest Content-Type", func() {
	It("uses the operation's declared media type", func() {
		conn := unifi.Local("127.0.0.1", "secret")
		op := Operation{App: unifi.AppNetwork, Method: http.MethodPost, Path: "/v1/upload", BodyMediaType: "application/octet-stream"}
		req, err := BuildRequest(context.Background(), conn, op, Values{Body: []byte("x")})
		Expect(err).NotTo(HaveOccurred())
		Expect(req.Header.Get("Content-Type")).To(Equal("application/octet-stream"))
	})

	It("defaults to application/json when a body is sent for an op the spec gave no media type", func() {
		conn := unifi.Local("127.0.0.1", "secret")
		op := Operation{App: unifi.AppNetwork, Method: http.MethodPost, Path: "/v1/x"}
		req, err := BuildRequest(context.Background(), conn, op, Values{Body: []byte("x")})
		Expect(err).NotTo(HaveOccurred())
		Expect(req.Header.Get("Content-Type")).To(Equal("application/json"))
	})
})
