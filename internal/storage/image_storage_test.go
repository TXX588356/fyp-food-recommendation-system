package storage

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("objectNameFromPublicURL", func() {
	It("extracts an object inside the configured bucket", func() {
		objectName, err := objectNameFromPublicURL(
			"http://localhost:9000/images/custom-meals/test.jpg",
			"http://localhost:9000",
			"images",
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(objectName).To(Equal("custom-meals/test.jpg"))
	})

	It("rejects a URL outside the configured bucket", func() {
		_, err := objectNameFromPublicURL(
			"http://example.com/images/custom-meals/test.jpg",
			"http://localhost:9000",
			"images",
		)

		Expect(err).To(HaveOccurred())
	})

	It("rejects the bucket URL without an object name", func() {
		_, err := objectNameFromPublicURL(
			"http://localhost:9000/images/",
			"http://localhost:9000",
			"images",
		)

		Expect(err).To(HaveOccurred())
	})
})
