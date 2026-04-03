package utils

import (
	"path/filepath"
)

func JoinPath(elem ...string) string {
	return filepath.Join(elem...)
}
