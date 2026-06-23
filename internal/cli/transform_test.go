package cli_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/internal/cli"
)

var _ = Describe("ApplyTransforms", func() {
	decode := func(b []byte) any {
		var v any
		Expect(json.Unmarshal(b, &v)).To(Succeed())
		return v
	}

	It("returns the body unchanged when no options are set", func() {
		body := []byte(`{"a":1}`)
		Expect(cli.ApplyTransforms(body, cli.RenderOptions{})).To(Equal(body))
	})

	It("returns non-JSON bodies unchanged", func() {
		body := []byte("<html>not json</html>")
		Expect(cli.ApplyTransforms(body, cli.RenderOptions{Redact: true})).To(Equal(body))
	})

	It("masks values under secret-like keys at any depth", func() {
		body := []byte(`{"name":"wg","vpn":{"presharedKey":"abc","port":51820},"apiKey":"k"}`)
		got := decode(cli.ApplyTransforms(body, cli.RenderOptions{Redact: true})).(map[string]any)
		Expect(got["name"]).To(Equal("wg"))
		Expect(got["apiKey"]).To(Equal("***"))
		vpn := got["vpn"].(map[string]any)
		Expect(vpn["presharedKey"]).To(Equal("***"))
		Expect(vpn["port"]).To(BeEquivalentTo(51820))
	})

	It("projects record fields from a page object's data, keeping the envelope", func() {
		body := []byte(`{"count":2,"data":[{"name":"a","vlanId":1,"secret":"x"},{"name":"b","vlanId":2}]}`)
		got := decode(cli.ApplyTransforms(body, cli.RenderOptions{Fields: []string{"name"}})).(map[string]any)
		Expect(got["count"]).To(BeEquivalentTo(2))
		data := got["data"].([]any)
		Expect(data).To(HaveLen(2))
		first := data[0].(map[string]any)
		Expect(first).To(HaveKeyWithValue("name", "a"))
		Expect(first).NotTo(HaveKey("vlanId"))
		Expect(first).NotTo(HaveKey("secret"))
	})

	It("projects nested dot-paths from each record", func() {
		body := []byte(`[{"name":"p","action":{"type":"ALLOW","x":1}}]`)
		got := decode(cli.ApplyTransforms(body, cli.RenderOptions{Fields: []string{"action.type"}})).([]any)
		rec := got[0].(map[string]any)
		Expect(rec).NotTo(HaveKey("name"))
		Expect(rec["action"].(map[string]any)).To(Equal(map[string]any{"type": "ALLOW"}))
	})

	It("projects fields from a plain object", func() {
		body := []byte(`{"applicationVersion":"10.4.57","extra":"y"}`)
		got := decode(cli.ApplyTransforms(body, cli.RenderOptions{Fields: []string{"applicationVersion"}})).(map[string]any)
		Expect(got).To(Equal(map[string]any{"applicationVersion": "10.4.57"}))
	})

	It("limits a page object's data array", func() {
		body := []byte(`{"count":3,"data":[{"n":1},{"n":2},{"n":3}]}`)
		got := decode(cli.ApplyTransforms(body, cli.RenderOptions{Limit: 2})).(map[string]any)
		Expect(got["data"].([]any)).To(HaveLen(2))
	})

	It("limits a top-level array", func() {
		body := []byte(`[{"n":1},{"n":2},{"n":3}]`)
		got := decode(cli.ApplyTransforms(body, cli.RenderOptions{Limit: 1})).([]any)
		Expect(got).To(HaveLen(1))
	})

	It("composes limit, fields, and redact", func() {
		body := []byte(`{"count":3,"data":[{"name":"a","psk":"s1"},{"name":"b","psk":"s2"},{"name":"c"}]}`)
		got := decode(cli.ApplyTransforms(body, cli.RenderOptions{Limit: 2, Fields: []string{"name", "psk"}, Redact: true})).(map[string]any)
		data := got["data"].([]any)
		Expect(data).To(HaveLen(2))
		first := data[0].(map[string]any)
		Expect(first["name"]).To(Equal("a"))
		Expect(first["psk"]).To(Equal("***"))
	})
})
