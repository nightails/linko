package main

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
)

type closeFunc func() error

func initializeLogger(logFile string) (*slog.Logger, closeFunc, error) {
	debugHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	closeLogger := func() error {
		return nil
	}

	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open log file: %w", err)
		}
		bufferedFile := bufio.NewWriterSize(file, 8192)
		multiWriter := io.MultiWriter(os.Stderr, bufferedFile)

		infoHandler := slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})

		closeLogger = func() error {
			if err := bufferedFile.Flush(); err != nil {
				return fmt.Errorf("failed to flush buffered log file: %w", err)
			}
			if err := file.Close(); err != nil {
				return fmt.Errorf("failed to close log file: %w", err)
			}
			return nil
		}

		return slog.New(slog.NewMultiHandler(debugHandler, infoHandler)), closeLogger, nil
	}

	return slog.New(debugHandler), closeLogger, nil
}
