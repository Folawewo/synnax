package alamos_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/alamos"
)

var _ = Describe("experiment", func() {
	Describe("Creating a new experiment", func() {
		It("Should create the experiment without panicking", func() {
			Expect(func() { alamos.New("test") }).ToNot(Panic())
		})
	})
})
