package main

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSpecgen(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "specgen suite")
}
