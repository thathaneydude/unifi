package unifi_test

import (
	"errors"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/unifi"
)

var _ = Describe("APIError", func() {
	It("parses the UniFi error envelope", func() {
		body := []byte(`{"statusCode":404,"statusName":"NOT_FOUND","message":"missing"}`)
		err := unifi.NewAPIError("getSite", http.StatusNotFound, body)
		Expect(err.Status).To(Equal(404))
		Expect(err.Message).To(Equal("missing"))
		Expect(err.Error()).To(ContainSubstring("getSite"))
		Expect(err.Error()).To(ContainSubstring("404"))
	})

	It("falls back to the raw body when not JSON", func() {
		err := unifi.NewAPIError("op", 500, []byte("boom"))
		Expect(err.Message).To(Equal("boom"))
	})

	It("sets the Operation and Body fields", func() {
		body := []byte(`{"statusCode":403,"message":"forbidden"}`)
		err := unifi.NewAPIError("createDevice", http.StatusForbidden, body)
		Expect(err.Operation).To(Equal("createDevice"))
		Expect(err.Body).To(Equal(body))
		Expect(err.Status).To(Equal(403))
	})

	It("implements the error interface", func() {
		err := unifi.NewAPIError("op", 500, []byte("fail"))
		var apiErr *unifi.APIError
		Expect(errors.As(err, &apiErr)).To(BeTrue())
	})

	It("produces a descriptive error string", func() {
		err := unifi.NewAPIError("listSites", 401, []byte(`{"message":"unauthorized"}`))
		Expect(err.Error()).To(ContainSubstring("listSites"))
		Expect(err.Error()).To(ContainSubstring("401"))
		Expect(err.Error()).To(ContainSubstring("unauthorized"))
	})
})
