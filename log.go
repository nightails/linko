package main

import (
	"fmt"
	"io"
	"log"
	"os"
)

func initializeLogger(logFile string) (*log.Logger, error) {
	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		multiWriter := io.MultiWriter(os.Stderr, file)
		return log.New(multiWriter, "", log.LstdFlags), nil
	}
	return log.New(os.Stderr, "", log.LstdFlags), nil
}
