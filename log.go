package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"boot.dev/linko/internal/build"
	"boot.dev/linko/internal/linkoerr"
	pkgerr "github.com/pkg/errors"
)

const logContextKey contextKey = "log_context"

type LogContext struct {
	Username string
}

type closeFunc func() error

func initializeLogger(logFile string) (*slog.Logger, closeFunc, error) {
	// default debug handler
	debugHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		ReplaceAttr: replaceAttr,
	})

	closeLogger := func() error {
		return nil
	}
	logger := slog.New(debugHandler)

	// open log file
	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open log file: %w", err)
		}
		bufferedFile := bufio.NewWriterSize(file, 8192)
		multiWriter := io.MultiWriter(os.Stderr, bufferedFile)

		infoHandler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			ReplaceAttr: replaceAttr,
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

		logger = slog.New(slog.NewMultiHandler(debugHandler, infoHandler))
	}

	env := os.Getenv("ENV")
	hostname, _ := os.Hostname()

	// add build info
	logger = logger.With(
		slog.String("git_sha", build.GitSHA),
		slog.String("build_time", build.BuildTime),
		slog.String("env", env),
		slog.String("hostname", hostname),
	)

	return logger, closeLogger, nil
}

type stackTracer interface {
	error
	StackTrace() pkgerr.StackTrace
}

type multiError interface {
	error
	Unwrap() []error
}

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == "error" {
		err, ok := a.Value.Any().(error)
		if !ok {
			return a
		}

		if multiErr, ok := errors.AsType[multiError](err); ok {
			errorAttrs := make([]slog.Attr, 0, len(multiErr.Unwrap()))

			for i, childErr := range multiErr.Unwrap() {
				childAttrs := []slog.Attr{
					{
						Key:   "message",
						Value: slog.StringValue(childErr.Error()),
					},
				}

				childAttrs = append(childAttrs, linkoerr.Attrs(childErr)...)

				errorAttrs = append(errorAttrs, slog.GroupAttrs(
					fmt.Sprintf("error_%d", i+1),
					childAttrs...,
				))
			}

			return slog.GroupAttrs("errors", errorAttrs...)
		}

		attrs := []slog.Attr{
			{
				Key:   "message",
				Value: slog.StringValue(err.Error()),
			},
		}

		if stackErr, ok := errors.AsType[stackTracer](err); ok {
			attrs = append(attrs, slog.Attr{
				Key:   "stack_trace",
				Value: slog.StringValue(fmt.Sprintf("%+v", stackErr.StackTrace())),
			})
		}

		attrs = append(attrs, linkoerr.Attrs(err)...)

		return slog.GroupAttrs("error", attrs...)
	}
	return a
}
