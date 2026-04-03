package storage

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"s3/types"
	"s3/utils"
)

type DiskStorage struct {
	dataDir string
	tmpDir  string
}

func NewDiskStorage(dataDir string) *DiskStorage {
	tmpDir := filepath.Join(dataDir, ".tmp")
	return &DiskStorage{
		dataDir: dataDir,
		tmpDir:  tmpDir,
	}
}

func (s *DiskStorage) Put(bucket, key string, body io.Reader) (string, error) {
	if err := utils.ValidateBucketName(bucket); err != nil {
		return "", err
	}
	if err := utils.ValidateKey(key); err != nil {
		return "", err
	}

	bucketDir := filepath.Join(s.dataDir, bucket)
	if err := os.MkdirAll(bucketDir, 0755); err != nil {
		return "", err
	}

	if err := os.MkdirAll(s.tmpDir, 0755); err != nil {
		return "", err
	}

	tmpFile, err := os.CreateTemp(s.tmpDir, "upload-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()

	written, err := io.Copy(tmpFile, body)
	if err != nil {
		os.Remove(tmpPath)
		tmpFile.Close()
		return "", err
	}
	_ = written
	tmpFile.Close()

	clean, err := utils.SafePath(s.dataDir, bucket, key)
	if err != nil {
		os.Remove(tmpPath)
		return "", err
	}

	targetDir := filepath.Dir(clean)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		os.Remove(tmpPath)
		return "", err
	}

	if err := os.Rename(tmpPath, clean); err != nil {
		os.Remove(tmpPath)
		return "", err
	}

	etag, err := s.calculateETagForPath(clean)
	return etag, err
}

func (s *DiskStorage) Get(bucket, key string) (io.ReadSeeker, int64, time.Time, string, error) {
	clean, err := utils.SafePath(s.dataDir, bucket, key)
	if err != nil {
		return nil, 0, time.Time{}, "", err
	}

	file, err := os.Open(clean)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, time.Time{}, "", types.ErrObjectNotFound
		}
		return nil, 0, time.Time{}, "", err
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, 0, time.Time{}, "", err
	}

	etag, err := s.calculateETagForPath(clean)
	if err != nil {
		file.Close()
		return nil, 0, time.Time{}, "", err
	}

	return file, stat.Size(), stat.ModTime(), etag, nil
}

func (s *DiskStorage) Delete(bucket, key string) error {
	clean, err := utils.SafePath(s.dataDir, bucket, key)
	if err != nil {
		return err
	}

	err = os.Remove(clean)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *DiskStorage) Head(bucket, key string) (int64, time.Time, string, error) {
	clean, err := utils.SafePath(s.dataDir, bucket, key)
	if err != nil {
		return 0, time.Time{}, "", err
	}

	stat, err := os.Stat(clean)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, time.Time{}, "", types.ErrObjectNotFound
		}
		return 0, time.Time{}, "", err
	}

	etag, err := s.calculateETagForPath(clean)
	if err != nil {
		return 0, time.Time{}, "", err
	}

	return stat.Size(), stat.ModTime(), etag, nil
}

func (s *DiskStorage) List(bucket, prefix, delimiter string) (*types.ListResult, error) {
	bucketDir := filepath.Join(s.dataDir, bucket)
	if _, err := os.Stat(bucketDir); os.IsNotExist(err) {
		return nil, types.ErrBucketNotFound
	}

	result := &types.ListResult{
		Objects:        []types.ObjectInfo{},
		CommonPrefixes: []string{},
	}

	err := filepath.Walk(bucketDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(bucketDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		if prefix != "" && !hasPrefix(relPath, prefix) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() && delimiter != "" {
			relativeWithoutPrefix := relPath
			if prefix != "" {
				relativeWithoutPrefix = relPath[len(prefix):]
			}
			if containsDelimiter(relativeWithoutPrefix, delimiter) {
				dirPath := getPrefixBeforeDelimiter(relativeWithoutPrefix, delimiter)
				fullPrefix := prefix + dirPath + delimiter
				if !containsString(result.CommonPrefixes, fullPrefix) {
					result.CommonPrefixes = append(result.CommonPrefixes, fullPrefix)
				}
				return filepath.SkipDir
			}
		}

		if !info.IsDir() {
			key := relPath
			obj := types.ObjectInfo{
				Key:          key,
				Size:         info.Size(),
				LastModified: info.ModTime(),
				ETag:         "",
			}
			result.Objects = append(result.Objects, obj)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *DiskStorage) DeleteBucket(bucket string) error {
	bucketDir := filepath.Join(s.dataDir, bucket)
	err := os.RemoveAll(bucketDir)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *DiskStorage) BucketExists(bucket string) (bool, error) {
	bucketDir := filepath.Join(s.dataDir, bucket)
	_, err := os.Stat(bucketDir)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (s *DiskStorage) calculateETagForPath(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	return CalculateETag(file)
}

func hasPrefix(path, prefix string) bool {
	if prefix == "" {
		return true
	}
	return len(path) >= len(prefix) && path[:len(prefix)] == prefix
}

func containsDelimiter(path, delimiter string) bool {
	for i := 0; i < len(path)-len(delimiter); i++ {
		if path[i:i+len(delimiter)] == delimiter {
			return true
		}
	}
	return false
}

func getPrefixBeforeDelimiter(path, delimiter string) string {
	for i := 0; i < len(path)-len(delimiter); i++ {
		if path[i:i+len(delimiter)] == delimiter {
			return path[:i]
		}
	}
	return path
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
