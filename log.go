package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"boot.dev/linko/internal/build"
	"boot.dev/linko/internal/linkoerr"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	pkgerr "github.com/pkg/errors"
)

const logContextKey contextKey = "log_context"

type LogContext struct {
	Username string
	Error    error
}

type closeFunc func() error

func initializeLogger(logFile string) (*slog.Logger, closeFunc, error) {
	var (
		handlers []slog.Handler
		closers  []closeFunc
	)

	replaceAttr := func(groups []string, a slog.Attr) slog.Attr {
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

	// console logger
	w := os.Stderr
	debugHandler := tint.NewHandler(w, &tint.Options{
		Level:       slog.LevelDebug,
		ReplaceAttr: replaceAttr,
		NoColor:     !isatty.IsTerminal(w.Fd()) || isatty.IsCygwinTerminal(w.Fd()),
	})
	handlers = append(handlers, debugHandler)

	// open log file
	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0x644)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open log file: %w", err)
		}
		bufferedFile := bufio.NewWriter(file)
		handlers = append(handlers, slog.NewJSONHandler(bufferedFile, &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			ReplaceAttr: replaceAttr,
		}))
		closers = append(closers, func() error {
			if err := bufferedFile.Flush(); err != nil {
				return fmt.Errorf("failed to flush buffered log file: %w", err)
			}
			if err := file.Close(); err != nil {
				return fmt.Errorf("failed to close log file: %w", err)
			}
			return nil
		})

		closers = append(closers, func() error {
			if err := bufferedFile.Flush(); err != nil {
				return fmt.Errorf("failed to flush buffered log file: %w", err)
			}
			if err := file.Close(); err != nil {
				return fmt.Errorf("failed to close log file: %w", err)
			}
			return nil
		})
	}

	cleanup := func() error {
		var errs []error
		for _, closer := range closers {
			errs = append(errs, closer())
		}
		return errors.Join(errs...)
	}

	logger := slog.New(slog.NewMultiHandler(handlers...))

	env := os.Getenv("ENV")
	hostname, _ := os.Hostname()

	// add build info
	logger = logger.With(
		slog.String("git_sha", build.GitSHA),
		slog.String("build_time", build.BuildTime),
		slog.String("env", env),
		slog.String("hostname", hostname),
	)

	return logger, cleanup, nil
}

type stackTracer interface {
	error
	StackTrace() pkgerr.StackTrace
}

type multiError interface {
	error
	Unwrap() []error
}

func httpError(ctx context.Context, w http.ResponseWriter, status int, err error) {
	if logCtx, ok := ctx.Value(logContextKey).(*LogContext); ok {
		logCtx.Error = err
	}
	http.Error(w, err.Error(), status)
}
