package cli

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("synthesizeOperationID", func() {
	DescribeTable("mirrors oapi-codegen method+path naming",
		func(method, path, want string) {
			Expect(synthesizeOperationID(method, path)).To(Equal(want))
		},
		Entry("simple collection", "GET", "/v1/alarm-hubs", "GetV1AlarmHubs"),
		Entry("path param", "GET", "/v1/alarm-hubs/{id}", "GetV1AlarmHubsId"),
		Entry("nested params", "POST", "/v1/alarm-hubs/{id}/outputs/{outputId}/trigger", "PostV1AlarmHubsIdOutputsOutputIdTrigger"),
		Entry("hyphenated segment", "DELETE", "/v1/arm-profiles/{id}", "DeleteV1ArmProfilesId"),
	)
})
