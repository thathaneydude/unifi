package unifi_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/unifi"
)

type sampleEvent struct {
	Type string `json:"type"`
}

var _ = Describe("Decode", func() {
	It("unmarshals a frame into a typed value", func() {
		evt, err := unifi.Decode[sampleEvent]([]byte(`{"type":"motion"}`))
		Expect(err).NotTo(HaveOccurred())
		Expect(evt.Type).To(Equal("motion"))
	})

	It("returns an error for malformed JSON", func() {
		_, err := unifi.Decode[sampleEvent]([]byte(`not json`))
		Expect(err).To(HaveOccurred())
	})

	It("handles nested structs", func() {
		type nested struct {
			Outer struct {
				Inner string `json:"inner"`
			} `json:"outer"`
		}
		v, err := unifi.Decode[nested]([]byte(`{"outer":{"inner":"hello"}}`))
		Expect(err).NotTo(HaveOccurred())
		Expect(v.Outer.Inner).To(Equal("hello"))
	})

	It("handles an array of typed events", func() {
		type event struct {
			ID int `json:"id"`
		}
		frames := [][]byte{
			[]byte(`{"id":1}`),
			[]byte(`{"id":2}`),
			[]byte(`{"id":3}`),
		}
		for i, f := range frames {
			evt, err := unifi.Decode[event](f)
			Expect(err).NotTo(HaveOccurred())
			Expect(evt.ID).To(Equal(i + 1))
		}
	})
})
