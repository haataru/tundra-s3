package types

import (
	"errors"
	"io"
	"time"
)

var (
	ErrBucketNotFound = errors.New("bucket not found")
	ErrObjectNotFound = errors.New("object not found")
	ErrInvalidBucket  = errors.New("invalid bucket name")
	ErrInvalidKey     = errors.New("invalid key")
	ErrKeyTooLong     = errors.New("key too long (max 1024)")
	ErrPathTraversal  = errors.New("path traversal detected")
	ErrBucketTooLong  = errors.New("bucket name too long (max 63)")
	ErrBucketTooShort = errors.New("bucket name too short (min 3)")
)

type ObjectInfo struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"lastModified"`
	ETag         string    `json:"etag"`
}

type ListResult struct {
	Objects        []ObjectInfo `json:"objects"`
	CommonPrefixes []string     `json:"commonPrefixes,omitempty"`
}

type Storage interface {
	Put(bucket, key string, body io.Reader) (string, error)
	Get(bucket, key string) (io.ReadSeeker, int64, time.Time, string, error)
	Delete(bucket, key string) error
	Head(bucket, key string) (int64, time.Time, string, error)
	List(bucket, prefix, delimiter string) (*ListResult, error)
	DeleteBucket(bucket string) error
	BucketExists(bucket string) (bool, error)
}

type Config struct {
	Port       string
	DataDir    string
	MaxConcurr int
}
