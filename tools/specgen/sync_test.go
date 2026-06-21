package main

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validate", func() {
	It("accepts a valid augmented network spec", func() {
		augmented, err := Augment("network", "v10.3.58", minimalNetworkSpec)
		Expect(err).NotTo(HaveOccurred())
		Expect(Validate(augmented)).To(Succeed())
	})

	It("rejects a minimal spec missing info and paths", func() {
		bare := []byte(`{"openapi":"3.1.0"}`)
		err := Validate(bare)
		Expect(err).To(HaveOccurred())
	})
})

// minimalNetworkSpec is a small Network-like OpenAPI 3.1.0 spec for testing Validate.
var minimalNetworkSpec = []byte(`{
  "openapi": "3.1.0",
  "info": {
    "title": "UniFi Network",
    "version": "0.0.0"
  },
  "paths": {
    "/v1/devices": {
      "get": {
        "summary": "List devices",
        "operationId": "listDevices",
        "responses": {
          "200": { "description": "OK" }
        }
      }
    }
  },
  "components": {}
}`)
