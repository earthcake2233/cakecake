package storage

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// OSS wraps a bucket handle for uploads.
type OSS struct {
	bucket *oss.Bucket
}

// NewOSS builds a client. endpoint must be the regional host only, e.g.
// https://oss-cn-beijing.aliyuncs.com — not https://bucket.oss-cn-beijing.aliyuncs.com
// (the SDK prepends the bucket name for virtual-hosted requests).
func NewOSS(endpoint, accessKeyID, accessKeySecret, bucketName string) (*OSS, error) {
	if endpoint == "" || accessKeyID == "" || accessKeySecret == "" || bucketName == "" {
		return nil, fmt.Errorf("oss configuration incomplete")
	}
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, err
	}
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return nil, err
	}
	return &OSS{bucket: bucket}, nil
}

// UploadFile uploads a local file to objectKey.
func (o *OSS) UploadFile(objectKey, localPath string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return o.bucket.PutObject(objectKey, f)
}

// UploadReader uploads from an io.Reader.
func (o *OSS) UploadReader(objectKey string, r io.Reader) error {
	return o.bucket.PutObject(objectKey, r)
}

// DeleteObject removes one object from the bucket. Missing keys are ignored.
func (o *OSS) DeleteObject(objectKey string) error {
	key := strings.TrimPrefix(strings.TrimSpace(objectKey), "/")
	if key == "" {
		return nil
	}
	if err := o.bucket.DeleteObject(key); err != nil {
		if ossErr, ok := err.(oss.ServiceError); ok && ossErr.StatusCode == 404 {
			return nil
		}
		return err
	}
	return nil
}

// DeleteObjects removes multiple objects; empty keys are skipped.
func (o *OSS) DeleteObjects(objectKeys []string) error {
	keys := make([]string, 0, len(objectKeys))
	for _, k := range objectKeys {
		k = strings.TrimPrefix(strings.TrimSpace(k), "/")
		if k != "" {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	_, err := o.bucket.DeleteObjects(keys)
	if err != nil {
		return err
	}
	return nil
}
