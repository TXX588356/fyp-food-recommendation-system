package storage

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

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

var _ = Describe("ImageStorage", func() {
	newTestStorage := func(bucket string) (*ImageStorage, *string, *string, *string) {
		var requestPath string
		var uploadedContentType string
		var uploadedBody string

		client, err := minio.New("object-storage.test", &minio.Options{
			Creds:        credentials.NewStaticV4("access-key", "secret-key", ""),
			Secure:       false,
			BucketLookup: minio.BucketLookupPath,
			Region:       "us-east-1",
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				statusCode := http.StatusNotFound
				switch r.Method {
				case http.MethodPut:
					requestPath = r.URL.Path
					uploadedContentType = r.Header.Get("Content-Type")
					body, err := io.ReadAll(r.Body)
					Expect(err).NotTo(HaveOccurred())
					uploadedBody = string(body)
					statusCode = http.StatusOK
				case http.MethodDelete:
					requestPath = r.URL.Path
					statusCode = http.StatusNoContent
				}

				return &http.Response{
					StatusCode: statusCode,
					Header:     http.Header{},
					Body:       io.NopCloser(strings.NewReader("")),
					Request:    r,
				}, nil
			}),
		})
		Expect(err).NotTo(HaveOccurred())

		return &ImageStorage{
			client:        client,
			bucket:        bucket,
			publicBaseURL: "http://public.test",
		}, &requestPath, &uploadedContentType, &uploadedBody
	}

	It("maps supported image content types to file extensions", func() {
		extension, err := extensionForContentType("image/jpeg")
		Expect(err).NotTo(HaveOccurred())
		Expect(extension).To(Equal(".jpg"))

		extension, err = extensionForContentType("image/png")
		Expect(err).NotTo(HaveOccurred())
		Expect(extension).To(Equal(".png"))
	})

	It("rejects unsupported image content types before uploading", func() {
		storage := &ImageStorage{}

		objectName, publicURL, err := storage.UploadMealImage(
			context.Background(),
			strings.NewReader("not-an-image"),
			int64(len("not-an-image")),
			"image/gif",
		)

		Expect(err).To(MatchError("unsupported image type: image/gif"))
		Expect(objectName).To(BeEmpty())
		Expect(publicURL).To(BeEmpty())
	})

	It("does nothing when deleting an empty object name", func() {
		storage := &ImageStorage{}

		err := storage.DeleteObject(context.Background(), "")

		Expect(err).NotTo(HaveOccurred())
	})

	It("returns an error for invalid object storage endpoint", func() {
		storage, err := NewImageStorage(
			context.Background(),
			"http://object-storage.test",
			"access-key",
			"secret-key",
			"images",
			false,
			"http://public.test",
		)

		Expect(err).To(MatchError(ContainSubstring("create object storage client")))
		Expect(storage).To(BeNil())
	})

	It("uploads a supported meal image and returns object name with public URL", func() {
		storage, requestPath, uploadedContentType, uploadedBody := newTestStorage("images")

		objectName, publicURL, err := storage.UploadMealImage(
			context.Background(),
			strings.NewReader("png-bytes"),
			int64(len("png-bytes")),
			"image/png",
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(objectName).To(HavePrefix("custom-meals/"))
		Expect(objectName).To(HaveSuffix(".png"))
		Expect(publicURL).To(Equal("http://public.test/images/" + objectName))
		Expect(*requestPath).To(Equal("/images/" + objectName))
		Expect(*uploadedContentType).To(Equal("image/png"))
		Expect(*uploadedBody).To(ContainSubstring("png-bytes"))
	})

	It("deletes a meal image using its public URL", func() {
		storage, requestPath, _, _ := newTestStorage("images")

		err := storage.DeleteMealImage(context.Background(), "http://public.test/images/custom-meals/test.jpg")

		Expect(err).NotTo(HaveOccurred())
		Expect(*requestPath).To(Equal("/images/custom-meals/test.jpg"))
	})

	It("rejects deletion for image URLs outside the configured bucket", func() {
		storage, requestPath, _, _ := newTestStorage("images")

		err := storage.DeleteMealImage(context.Background(), "http://example.com/images/custom-meals/test.jpg")

		Expect(err).To(MatchError("image URL does not belong to configured object storage bucket"))
		Expect(*requestPath).To(BeEmpty())
	})

	It("presigns a meal image URL in the configured bucket", func() {
		storage, _, _, _ := newTestStorage("images")

		presignedURL, err := storage.PresignMealImage(context.Background(), "http://public.test/images/custom-meals/test.jpg")

		Expect(err).NotTo(HaveOccurred())
		Expect(presignedURL).To(HavePrefix("http://object-storage.test/images/custom-meals/test.jpg?"))
		Expect(presignedURL).To(ContainSubstring("X-Amz-Signature="))
	})
})
