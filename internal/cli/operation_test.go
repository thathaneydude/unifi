package cli_test

import (
	"os"

	"github.com/getkin/kin-openapi/openapi3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/internal/cli"
	"github.com/thathaneydude/unifi/unifi"
)

func loadFixture() *openapi3.T { return loadNamedFixture("testdata/fixture.openapi.json") }

func loadNamedFixture(path string) *openapi3.T {
	data, err := os.ReadFile(path)
	Expect(err).NotTo(HaveOccurred())
	doc, err := openapi3.NewLoader().LoadFromData(data)
	Expect(err).NotTo(HaveOccurred())
	return doc
}

var _ = Describe("Operations", func() {
	It("extracts operations with params and body flags", func() {
		ops := cli.OperationsFor(loadFixture(), unifi.AppNetwork, "v1.0.0")

		byID := map[string]cli.Operation{}
		for _, op := range ops {
			byID[op.ID] = op
		}

		Expect(byID).To(HaveKey("ListWidgets"))
		Expect(byID["ListWidgets"].Method).To(Equal("GET"))
		Expect(byID["ListWidgets"].QueryParams).To(HaveLen(1))
		Expect(byID["ListWidgets"].QueryParams[0].Name).To(Equal("limit"))

		Expect(byID["GetWidget"].Path).To(Equal("/v1/widgets/{id}"))
		Expect(byID["GetWidget"].PathParams[0].Name).To(Equal("id"))
		Expect(byID["GetWidget"].PathParams[0].Required).To(BeTrue())

		Expect(byID["CreateWidget"].Method).To(Equal("POST"))
		Expect(byID["CreateWidget"].HasBody()).To(BeTrue())
		Expect(byID["CreateWidget"].BodyMediaType).To(Equal("application/json"))
		Expect(byID["CreateWidget"].Mutating()).To(BeTrue())
		Expect(byID["ListWidgets"].Mutating()).To(BeFalse())
	})

	It("synthesizes ids for specs that omit operationId (Protect)", func() {
		cat, err := cli.LoadCatalog()
		Expect(err).NotTo(HaveOccurred())
		doc, version, err := cat.Doc(unifi.AppProtect, "")
		Expect(err).NotTo(HaveOccurred())

		ops := cli.OperationsFor(doc, unifi.AppProtect, version)
		Expect(len(ops)).To(BeNumerically(">", 0))

		ids := map[string]bool{}
		for _, op := range ops {
			ids[op.ID] = true
		}
		Expect(ids).To(HaveKey("GetV1AlarmHubs"))
		Expect(ids).To(HaveKey("GetV1AlarmHubsId"))
	})

	Context("edge cases", func() {
		var byID map[string]cli.Operation

		BeforeEach(func() {
			byID = map[string]cli.Operation{}
			for _, op := range cli.OperationsFor(loadNamedFixture("testdata/edgecases.openapi.json"), unifi.AppNetwork, "v1.0.0") {
				byID[op.ID] = op
			}
		})

		It("merges path-item-level params and lets the operation override them", func() {
			op := byID["GetThing"]
			// id comes only from the path-item level.
			Expect(op.PathParams).To(HaveLen(1))
			Expect(op.PathParams[0].Name).To(Equal("id"))
			// shared is defined at both levels; the operation-level definition wins.
			Expect(op.QueryParams).To(HaveLen(1))
			Expect(op.QueryParams[0].Name).To(Equal("shared"))
			Expect(op.QueryParams[0].Required).To(BeTrue())
			Expect(op.QueryParams[0].Description).To(Equal("operation override"))
		})

		It("derives the body media type, falling back past JSON when absent", func() {
			Expect(byID["UploadBlob"].BodyMediaType).To(Equal("application/octet-stream"))
			Expect(byID["MultiBody"].BodyMediaType).To(Equal("application/json"))
		})

		It("disambiguates colliding synthesized operation ids deterministically", func() {
			// /v1/foo-bar and /v1/foo_bar both synthesize to GetV1FooBar.
			Expect(byID).To(HaveKey("GetV1FooBar"))
			Expect(byID).To(HaveKey("GetV1FooBar-2"))
		})

		It("keeps a path and query param sharing a name as distinct params", func() {
			op := byID["PutItem"]
			Expect(op.PathParams).To(HaveLen(1))
			Expect(op.PathParams[0].Name).To(Equal("id"))
			Expect(op.QueryParams).To(HaveLen(1))
			Expect(op.QueryParams[0].Name).To(Equal("id"))
		})
	})
})
