package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"

	"s3/server"
	"s3/storage"
)

func main() {
	port := flag.String("port", "8080", "Server port (default: 8080)")
	dataDir := flag.String("data-dir", os.Getenv("DATA_DIR"), "Data directory (default: ./data)")
	maxConcurr := flag.Int("max-concurr", 500, "Maximum concurrent requests")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if *dataDir == "" {
		exePath, _ := os.Executable()
		exeDir := filepath.Dir(exePath)
		*dataDir = filepath.Join(exeDir, "..", "data")
	}

	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		logger.Error("Failed to create data directory", "error", err)
		os.Exit(1)
	}

	diskStorage := storage.NewDiskStorage(*dataDir)

	logger.Info("Starting S3-like storage", "port", *port, "data-dir", *dataDir)

	srv := server.New(":"+*port, diskStorage, *maxConcurr, logger)

	if err := srv.ListenAndServe(); err != nil {
		logger.Error("Server error", "error", err)
		os.Exit(1)
	}
}
