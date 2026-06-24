package interfaces

import (
	"context"
	"io"
)

type ImageStorage interface {
	UploadMealImage(ctx context.Context, reader io.Reader, size int64, contentType string) (objectName, publicURL string, err error)
	DeleteObject(ctx context.Context, objectName string) error
	DeleteMealImage(ctx context.Context, imageURL string) error
}
