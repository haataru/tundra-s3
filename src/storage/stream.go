package storage

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

func CalculateETag(file *os.File) (string, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	return "\"" + hex.EncodeToString(hash.Sum(nil)) + "\"", nil
}

func CopyToFile(dst *os.File, src io.Reader) (int64, error) {
	return io.Copy(dst, src)
}
