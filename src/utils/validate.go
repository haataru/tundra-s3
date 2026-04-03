package utils

import (
	"path/filepath"
	"regexp"
	"strings"

	"s3/types"
)

var bucketRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)

func ValidateBucketName(name string) error {
	if len(name) < 3 {
		return types.ErrBucketTooShort
	}
	if len(name) > 63 {
		return types.ErrBucketTooLong
	}
	if !bucketRegex.MatchString(name) {
		return types.ErrInvalidBucket
	}
	return nil
}

func ValidateKey(key string) error {
	if key == "" {
		return types.ErrInvalidKey
	}
	if len(key) > 1024 {
		return types.ErrKeyTooLong
	}
	if strings.Contains(key, "..") {
		return types.ErrPathTraversal
	}
	if strings.HasPrefix(key, "/") {
		return types.ErrInvalidKey
	}
	return nil
}

func SafePath(dataDir, bucket, key string) (string, error) {
	clean := filepath.Clean(filepath.Join(dataDir, bucket, key))
	if !strings.HasPrefix(clean, dataDir+string(filepath.Separator)) && clean != dataDir {
		return "", types.ErrPathTraversal
	}
	return clean, nil
}
