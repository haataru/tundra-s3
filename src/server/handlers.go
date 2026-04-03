package server

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"s3/types"
)

type handlers struct {
	storage types.Storage
	logger  *slog.Logger
}

func (h *handlers) Put(w http.ResponseWriter, r *http.Request) {
	bucket, key, ok := parseBucketKey(r)
	if !ok {
		http.Error(w, "Invalid bucket or key", http.StatusBadRequest)
		return
	}

	etag, err := h.storage.Put(bucket, key, r.Body)
	if err != nil {
		h.logger.Error("Put failed", "error", err)
		writeError(w, err)
		return
	}

	w.Header().Set("ETag", etag)
	w.WriteHeader(http.StatusOK)
}

func (h *handlers) Get(w http.ResponseWriter, r *http.Request) {
	bucket, key, ok := parseBucketKey(r)
	if !ok {
		http.Error(w, "Invalid bucket or key", http.StatusBadRequest)
		return
	}

	reader, size, modTime, etag, err := h.storage.Get(bucket, key)
	if err != nil {
		h.logger.Error("Get failed", "error", err)
		writeError(w, err)
		return
	}
	if closer, ok := reader.(io.Closer); ok {
		defer closer.Close()
	}
	_ = size

	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", modTime.UTC().Format(http.TimeFormat))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))

	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		w.WriteHeader(http.StatusOK)
		io.Copy(w, reader)
		return
	}

	rangeStart, rangeEnd, err := parseRange(rangeHeader, size)
	if err != nil {
		http.Error(w, "Invalid Range", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	if rangeEnd == 0 {
		rangeEnd = size - 1
	}

	w.Header().Set("Content-Range", formatContentRange(rangeStart, rangeEnd, size))
	w.Header().Set("Content-Length", strconv.FormatInt(rangeEnd-rangeStart+1, 10))
	w.WriteHeader(http.StatusPartialContent)

	reader.Seek(rangeStart, io.SeekStart)
	limitedReader := io.LimitedReader{R: reader, N: rangeEnd - rangeStart + 1}
	io.Copy(w, &limitedReader)
}

func parseRange(header string, size int64) (start, end int64, err error) {
	if !strings.HasPrefix(header, "bytes=") {
		return 0, 0, fmt.Errorf("invalid range format")
	}
	parts := strings.Split(header[6:], "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range format")
	}
	if parts[0] == "" {
		start = size - 1
	} else {
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, 0, err
		}
	}
	if parts[1] == "" {
		end = size - 1
	} else {
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, 0, err
		}
	}
	return start, end, nil
}

func formatContentRange(start, end, total int64) string {
	return fmt.Sprintf("bytes %d-%d/%d", start, end, total)
}

func (h *handlers) Delete(w http.ResponseWriter, r *http.Request) {
	bucket, key, ok := parseBucketKey(r)
	if !ok {
		http.Error(w, "Invalid bucket or key", http.StatusBadRequest)
		return
	}

	path := r.URL.Path
	isBucketDelete := path == "/"+bucket || path == "/"+bucket+"/" || key == ""

	if isBucketDelete {
		err := h.storage.DeleteBucket(bucket)
		if err != nil {
			h.logger.Error("Delete bucket failed", "error", err)
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	err := h.storage.Delete(bucket, key)
	if err != nil {
		h.logger.Error("Delete failed", "error", err)
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) Head(w http.ResponseWriter, r *http.Request) {
	bucket, key, ok := parseBucketKey(r)
	if !ok {
		http.Error(w, "Invalid bucket or key", http.StatusBadRequest)
		return
	}

	size, modTime, etag, err := h.storage.Head(bucket, key)
	if err != nil {
		h.logger.Error("Head failed", "error", err)
		writeError(w, err)
		return
	}

	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", modTime.UTC().Format(http.TimeFormat))
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Accept-Ranges", "bytes")
	w.WriteHeader(http.StatusOK)
}

func (h *handlers) List(w http.ResponseWriter, r *http.Request) {
	bucket := extractBucket(r)
	if bucket == "" {
		http.Error(w, "Bucket required", http.StatusBadRequest)
		return
	}

	prefix := r.URL.Query().Get("prefix")
	delimiter := r.URL.Query().Get("delimiter")

	result, err := h.storage.List(bucket, prefix, delimiter)
	if err != nil {
		h.logger.Error("List failed", "error", err)
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(formatListResult(result)))
}

func parseBucketKey(r *http.Request) (bucket, key string, ok bool) {
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 2)
	if len(parts) < 1 {
		return "", "", false
	}
	bucket = parts[0]
	if len(parts) > 1 {
		key = parts[1]
	}
	return bucket, key, true
}

func extractBucket(r *http.Request) string {
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 2)
	if len(parts) < 1 {
		return ""
	}
	return parts[0]
}

func writeError(w http.ResponseWriter, err error) {
	switch err {
	case types.ErrBucketNotFound:
		http.Error(w, "Bucket not found", http.StatusNotFound)
	case types.ErrObjectNotFound:
		http.Error(w, "Object not found", http.StatusNotFound)
	case types.ErrInvalidBucket:
		http.Error(w, "Invalid bucket name", http.StatusBadRequest)
	case types.ErrInvalidKey:
		http.Error(w, "Invalid key", http.StatusBadRequest)
	case types.ErrKeyTooLong:
		http.Error(w, "Key too long", http.StatusBadRequest)
	case types.ErrPathTraversal:
		http.Error(w, "Path traversal detected", http.StatusBadRequest)
	case types.ErrBucketTooLong:
		http.Error(w, "Bucket name too long", http.StatusBadRequest)
	case types.ErrBucketTooShort:
		http.Error(w, "Bucket name too short", http.StatusBadRequest)
	default:
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func formatListResult(result *types.ListResult) string {
	var sb strings.Builder
	sb.WriteString(`{"objects":[`)
	for i, obj := range result.Objects {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`{"key":"`)
		sb.WriteString(obj.Key)
		sb.WriteString(`","size":`)
		sb.WriteString(strconv.FormatInt(obj.Size, 10))
		sb.WriteString(`,"lastModified":"`)
		sb.WriteString(obj.LastModified.UTC().Format(time.RFC3339))
		sb.WriteString(`","etag":"`)
		sb.WriteString(obj.ETag)
		sb.WriteString(`"}`)
	}
	sb.WriteString("],\"commonPrefixes\":[")
	for i, cp := range result.CommonPrefixes {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`"`)
		sb.WriteString(cp)
		sb.WriteString(`"`)
	}
	sb.WriteString("]}")
	return sb.String()
}
