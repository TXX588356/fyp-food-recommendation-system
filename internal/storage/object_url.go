package storage

import (
	"net/url"
	"strings"
)

// ObjectURLResolver converts an object storage key to its public URL without
// contacting object storage. Object keys remain deployment-independent in the database.
type ObjectURLResolver struct {
	publicBaseURL string
	bucket        string
}

func NewObjectURLResolver(publicBaseURL, bucket string) *ObjectURLResolver {
	return &ObjectURLResolver{
		publicBaseURL: strings.TrimRight(strings.TrimSpace(publicBaseURL), "/"),
		bucket:        strings.Trim(strings.TrimSpace(bucket), "/"),
	}
}

func (r *ObjectURLResolver) Resolve(objectKey string) string {
	objectKey = strings.Trim(strings.TrimSpace(objectKey), "/")
	if objectKey == "" {
		return ""
	}

	parts := strings.Split(objectKey, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}

	return r.publicBaseURL + "/" + r.bucket + "/" + strings.Join(parts, "/")
}
