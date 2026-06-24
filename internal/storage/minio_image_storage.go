package storage

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOImageStorage struct {
	client        *minio.Client
	bucket        string
	publicBaseURL string
}

func NewMinIOImageStorage(ctx context.Context, endpoint, accessKey, secretKey, bucket string, useSSL bool, publicBaseURL string) (*MinIOImageStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create MinIO client: %w", err)
	}

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("check MinIO bucket: %w", err)
	}

	if !exists {
		return nil, fmt.Errorf("MinIO bucket %q does not exist", bucket)
	}

	return &MinIOImageStorage{
		client:        client,
		bucket:        bucket,
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"),
	}, nil
}

func (s *MinIOImageStorage) UploadMealImage(ctx context.Context, reader io.Reader, size int64, contentType string) (string, string, error) {
	extension, err := extensionForContentType(contentType)
	if err != nil {
		return "", "", err
	}

	objectName := path.Join("custom-meals", uuid.NewString()+extension)

	_, err = s.client.PutObject(ctx, s.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})

	if err != nil {
		return "", "", fmt.Errorf("upload image to MinIO: %w", err)
	}

	publicURL := fmt.Sprintf("%s/%s/%s", s.publicBaseURL, s.bucket, objectName)

	return objectName, publicURL, nil
}

func (s *MinIOImageStorage) DeleteObject(ctx context.Context, objectName string) error {
	if objectName == "" {
		return nil
	}

	if err := s.client.RemoveObject(ctx, s.bucket, objectName, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete MinIO object: %w", err)
	}

	return nil
}

func (s *MinIOImageStorage) DeleteMealImage(ctx context.Context, imageURL string) error {
	objectName, err := objectNameFromPublicURL(imageURL, s.publicBaseURL, s.bucket)
	if err != nil {
		return err
	}

	return s.DeleteObject(ctx, objectName)
}

func objectNameFromPublicURL(imageURL, publicBaseURL, bucket string) (string, error) {
	prefix := fmt.Sprintf("%s/%s/", strings.TrimRight(publicBaseURL, "/"), bucket)
	if !strings.HasPrefix(imageURL, prefix) {
		return "", fmt.Errorf("image URL does not belong to configured MinIO bucket")
	}

	objectName := strings.TrimPrefix(imageURL, prefix)
	if strings.TrimSpace(objectName) == "" {
		return "", fmt.Errorf("image URL does not contain an object name")
	}

	return objectName, nil
}

func extensionForContentType(contentType string) (string, error) {
	switch contentType {
	case "image/jpeg":
		return ".jpg", nil
	case "image/png":
		return ".png", nil
	default:
		return "", fmt.Errorf("unsupported image type: %s", contentType)
	}
}
