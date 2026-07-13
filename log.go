package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"slices"

	"boot.dev/linko/internal/build"
	"boot.dev/linko/internal/linkoerr"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	pkgerr "github.com/pkg/errors"
	"gopkg.in/natefinch/lumberjack.v2"
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
		var sensitiveKeys = []string{"user", "password", "key", "apiKey", "secret", "pin", "creditcardno"}

		if slices.Contains(sensitiveKeys, a.Key) {
			return slog.String(a.Key, "[REDACTED]")
		}

		if a.Key == "long_url" {
			u, err := url.Parse(a.Value.String())
			if err != nil {
				return a
			}
			if u.User != nil {
				u.User = url.UserPassword("REDACTED", "REDACTED")
			}
			return slog.String(a.Key, u.String())
		}

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

	// file logger
	if logFile != "" {
		fileLogger := &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    1,
			MaxAge:     28,
			MaxBackups: 10,
			LocalTime:  false,
			Compress:   true,
		}
		handlers = append(handlers, slog.NewJSONHandler(fileLogger, &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			ReplaceAttr: replaceAttr,
		}))
		closers = append(closers, func() error {
			if err := fileLogger.Close(); err != nil {
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

	// main logger
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
	if status == 401 || status == 403 || status == 500 {
		http.Error(w, http.StatusText(status), status)
		return
	}
	http.Error(w, err.Error(), status)
}
