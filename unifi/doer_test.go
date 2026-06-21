package unifi_test

import (
	"context"
	"io"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	networkv10_3_58 "github.com/thathaneydude/unifi/lib/network/v10_3_58"
	"github.com/thathaneydude/unifi/unifi/internalfakes"
)

var _ = Describe("HTTPDoer fake drives generated client offline", func() {
	var (
		fake      *internalfakes.FakeHTTPDoer
		netClient *networkv10_3_58.ClientWithResponses
	)

	BeforeEach(func() {
		fake = &internalfakes.FakeHTTPDoer{}

		// Stub the fake to return a valid GetInfo JSON response.
		fake.DoReturns(
			&http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": {"application/json"}},
				Body: io.NopCloser(strings.NewReader(
					`{"applicationVersion":"10.3.58"}`,
				)),
			},
			nil,
		)

		var err error
		netClient, err = networkv10_3_58.NewClientWithResponses(
			"https://unifi.local/proxy/network/integration",
			networkv10_3_58.WithHTTPClient(fake),
		)
		Expect(err).NotTo(HaveOccurred())
	})

	It("calls Do exactly once for GetInfoWithResponse", func() {
		resp, err := netClient.GetInfoWithResponse(context.Background())
		Expect(err).NotTo(HaveOccurred())

		// The fake recorded exactly one call.
		Expect(fake.DoCallCount()).To(Equal(1))

		// Inspect the request the generated client passed to the fake.
		req := fake.DoArgsForCall(0)
		Expect(req).NotTo(BeNil())
		Expect(req.URL.Host).To(Equal("unifi.local"))
		Expect(req.URL.Path).To(Equal("/proxy/network/integration/v1/info"))

		// The decoded response body reflects the stub.
		Expect(resp.StatusCode()).To(Equal(http.StatusOK))
		Expect(resp.JSON200).NotTo(BeNil())
		Expect(resp.JSON200.ApplicationVersion).To(Equal("10.3.58"))
	})

	It("records multiple calls when invoked repeatedly", func() {
		// Stub two calls so neither exhausts the single return value.
		fake.DoReturnsOnCall(1,
			&http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": {"application/json"}},
				Body: io.NopCloser(strings.NewReader(
					`{"applicationVersion":"10.3.59"}`,
				)),
			},
			nil,
		)

		_, err := netClient.GetInfoWithResponse(context.Background())
		Expect(err).NotTo(HaveOccurred())

		_, err = netClient.GetInfoWithResponse(context.Background())
		Expect(err).NotTo(HaveOccurred())

		Expect(fake.DoCallCount()).To(Equal(2))
	})
})
