package storage

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// OSS wraps a bucket handle for uploads.
type OSS struct {
	bucket *oss.Bucket
}

// ObjectMeta describes one listed object.
type ObjectMeta struct {
	Key          string
	LastModified time.Time
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
		return nil, fmt.Errorf("oss client init failed")
	}
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return nil, fmt.Errorf("oss bucket init failed")
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

// DownloadFile downloads objectKey into localPath.
func (o *OSS) DownloadFile(objectKey, localPath string) error {
	key := strings.TrimPrefix(strings.TrimSpace(objectKey), "/")
	if key == "" {
		return fmt.Errorf("empty object key")
	}
	rc, err := o.bucket.GetObject(key)
	if err != nil {
		return err
	}
	defer rc.Close()
	f, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, rc)
	return err
}

// Exists reports whether objectKey exists in the bucket.
func (o *OSS) Exists(objectKey string) (bool, error) {
	key := strings.TrimPrefix(strings.TrimSpace(objectKey), "/")
	if key == "" {
		return false, nil
	}
	return o.bucket.IsObjectExist(key)
}

// Size returns the object size in bytes via a HEAD request.
func (o *OSS) Size(objectKey string) (int64, error) {
	key := strings.TrimPrefix(strings.TrimSpace(objectKey), "/")
	if key == "" {
		return 0, fmt.Errorf("empty object key")
	}
	meta, err := o.bucket.GetObjectMeta(key)
	if err != nil {
		return 0, err
	}
	if v := meta.Get("Content-Length"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse content-length %q: %w", v, err)
		}
		return n, nil
	}
	return 0, fmt.Errorf("content-length missing for %s", key)
}

// ReadPrefix downloads the first n bytes of an object via a range request.
// It is the cheap way to validate content (e.g. cover magic bytes) without
// pulling the whole object into the API process.
func (o *OSS) ReadPrefix(objectKey string, n int64) ([]byte, error) {
	key := strings.TrimPrefix(strings.TrimSpace(objectKey), "/")
	if key == "" {
		return nil, fmt.Errorf("empty object key")
	}
	body, err := o.bucket.GetObject(key, oss.Range(0, n-1))
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return io.ReadAll(io.LimitReader(body, n))
}

// ListObjects returns object metadata under prefix (paged). maxKeys bounds
// each page; the total is capped so a single cleanup run cannot enumerate an
// unbounded bucket.
func (o *OSS) ListObjects(prefix string, maxKeys int) ([]ObjectMeta, error) {
	prefix = strings.TrimPrefix(strings.TrimSpace(prefix), "/")
	if prefix == "" {
		return nil, nil
	}
	if maxKeys <= 0 {
		maxKeys = 1000
	}
	const totalCap = 20000
	var out []ObjectMeta
	marker := ""
	for {
		opts := []oss.Option{oss.Prefix(prefix), oss.MaxKeys(maxKeys)}
		if marker != "" {
			opts = append(opts, oss.Marker(marker))
		}
		res, err := o.bucket.ListObjects(opts...)
		if err != nil {
			return nil, err
		}
		for _, obj := range res.Objects {
			out = append(out, ObjectMeta{Key: obj.Key, LastModified: obj.LastModified})
		}
		if !res.IsTruncated {
			break
		}
		marker = res.NextMarker
		if marker == "" {
			break
		}
		if len(out) >= totalCap {
			break
		}
	}
	return out, nil
}

// PresignPut returns a time-limited HTTP PUT URL for client-side direct
// upload. The browser uploads straight to OSS, bypassing the API server.
// contentType must match the Content-Type header the browser will send:
// OSS includes it in the signature (StringToSign), so signing with an empty
// type but uploading with e.g. image/jpeg yields SignatureDoesNotMatch.
func (o *OSS) PresignPut(objectKey string, expiry time.Duration, contentType string) (string, error) {
	key := strings.TrimPrefix(strings.TrimSpace(objectKey), "/")
	if key == "" {
		return "", fmt.Errorf("empty object key")
	}
	opts := []oss.Option{}
	if contentType != "" {
		opts = append(opts, oss.ContentType(contentType))
	}
	u, err := o.bucket.SignURL(key, oss.HTTPPut, int64(expiry.Seconds()), opts...)
	if err != nil {
		return "", err
	}
	return u, nil
}

// PresignGet returns a time-limited HTTP GET URL for a private object. It is
// used to preview draft media without proxying the bytes through the API
// server or making the object publicly readable.
func (o *OSS) PresignGet(objectKey string, expiry time.Duration) (string, error) {
	key := strings.TrimPrefix(strings.TrimSpace(objectKey), "/")
	if key == "" {
		return "", fmt.Errorf("empty object key")
	}
	u, err := o.bucket.SignURL(key, oss.HTTPGet, int64(expiry.Seconds()))
	if err != nil {
		return "", err
	}
	return u, nil
}

// CopyObject copies an existing object to a new key in the same bucket
// (server-side copy: no bytes traverse the API server).
func (o *OSS) CopyObject(srcKey, dstKey string) error {
	src := strings.TrimPrefix(strings.TrimSpace(srcKey), "/")
	dst := strings.TrimPrefix(strings.TrimSpace(dstKey), "/")
	if src == "" || dst == "" {
		return fmt.Errorf("copy object: empty key")
	}
	_, err := o.bucket.CopyObject(src, dst)
	return err
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
