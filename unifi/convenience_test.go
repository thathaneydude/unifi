package unifi_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	networklatest "github.com/thathaneydude/unifi/lib/network/v10_3_58"
	protectlatest "github.com/thathaneydude/unifi/lib/protect/v7_1_46"
	"github.com/thathaneydude/unifi/unifi"
)

var _ = Describe("Convenience", func() {
	It("returns non-nil clients of the correct generated types", func() {
		c := unifi.Local("h", "k")

		netClient, netErr := c.Network()
		Expect(netErr).NotTo(HaveOccurred())
		Expect(netClient).NotTo(BeNil())
		// Assert the concrete type is the pinned generated client.
		Expect(netClient).To(BeAssignableToTypeOf(&networklatest.ClientWithResponses{}))

		protClient, protErr := c.Protect()
		Expect(protErr).NotTo(HaveOccurred())
		Expect(protClient).NotTo(BeNil())
		Expect(protClient).To(BeAssignableToTypeOf(&protectlatest.ClientWithResponses{}))
	})
})
