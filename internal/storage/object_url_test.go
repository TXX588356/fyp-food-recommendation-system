package storage

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestStorage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Storage Suite")
}

var _ = Describe("ObjectURLResolver", func() {
	It("should escape each object key segment", func() {
		resolver := NewObjectURLResolver("http://localhost:9000/", "images")

		got := resolver.Resolve("catalog meals/meal one/image #1.jpg")

		Expect(got).To(Equal("http://localhost:9000/images/catalog%20meals/meal%20one/image%20%231.jpg"))
	})

	It("should return empty for empty key", func() {
		resolver := NewObjectURLResolver("http://localhost:9000", "images")

		Expect(resolver.Resolve("  ")).To(BeEmpty())
	})
})
