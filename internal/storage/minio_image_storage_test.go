package storage

import "testing"

func TestObjectNameFromPublicURL(t *testing.T) {
	t.Run("extracts an object inside the configured bucket", func(t *testing.T) {
		objectName, err := objectNameFromPublicURL(
			"http://localhost:9000/images/custom-meals/test.jpg",
			"http://localhost:9000",
			"images",
		)
		if err != nil {
			t.Fatalf("expected valid image URL, got %v", err)
		}
		if objectName != "custom-meals/test.jpg" {
			t.Fatalf("expected object name custom-meals/test.jpg, got %q", objectName)
		}
	})

	t.Run("rejects a URL outside the configured bucket", func(t *testing.T) {
		_, err := objectNameFromPublicURL(
			"http://example.com/images/custom-meals/test.jpg",
			"http://localhost:9000",
			"images",
		)
		if err == nil {
			t.Fatal("expected an outside image URL to be rejected")
		}
	})

	t.Run("rejects the bucket URL without an object name", func(t *testing.T) {
		_, err := objectNameFromPublicURL(
			"http://localhost:9000/images/",
			"http://localhost:9000",
			"images",
		)
		if err == nil {
			t.Fatal("expected an empty object name to be rejected")
		}
	})
}
