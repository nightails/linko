package main

import (
	"io"
	"log"
	"os"
)

func initializeLogger() *log.Logger {
	val, ok := os.LookupEnv("LINKO_LOG_FILE")
	if !ok {
		return log.New(os.Stderr, "", log.LstdFlags)
	}
	file, err := os.OpenFile(val, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return log.New(os.Stderr, "", log.LstdFlags)
	}
	multiWriter := io.MultiWriter(os.Stderr, file)
	return log.New(multiWriter, "", log.LstdFlags)
}
