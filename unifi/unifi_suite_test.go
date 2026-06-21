package unifi_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUnifi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "unifi suite")
}
