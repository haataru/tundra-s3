package server

import (
	"log/slog"
	"net/http"

	"s3/types"
)

func NewRouter(storage types.Storage, logger *slog.Logger) *http.ServeMux {
	h := &handlers{
		storage: storage,
		logger:  logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", routeRequest(h))
	return mux
}

func routeRequest(h *handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			h.Put(w, r)
		case http.MethodGet:
			if r.URL.Path == "/" || isListRequest(r) {
				h.List(w, r)
			} else {
				h.Get(w, r)
			}
		case http.MethodDelete:
			h.Delete(w, r)
		case http.MethodHead:
			h.Head(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func isListRequest(r *http.Request) bool {
	path := r.URL.Path
	if path == "/" {
		return false
	}
	parts := splitPath(path)
	if len(parts) == 1 {
		return true
	}
	if len(parts) == 2 && parts[1] == "" {
		return true
	}
	return false
}

func splitPath(path string) []string {
	if path == "/" {
		return []string{}
	}
	result := []string{}
	current := ""
	for _, c := range path[1:] {
		if c == '/' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
