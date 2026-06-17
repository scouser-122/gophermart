package logger

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/fabien-marty/slog-helpers/pkg/stacktrace"
)

// Initialize creates default slog logger with specified logging level
func Initialize(level string, textWriter io.Writer, jsonWriter io.Writer) {
	logLevel := slog.LevelInfo
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "error":
		logLevel = slog.LevelError
	case "warn":
		logLevel = slog.LevelWarn
	}
	baseTextHandler := slog.NewTextHandler(textWriter, &slog.HandlerOptions{
		Level: logLevel,
	})
	stackTextHandler := stacktrace.New(baseTextHandler, &stacktrace.Options{
		Mode: stacktrace.ModePrint,
	})
	var logger *slog.Logger
	if jsonWriter == nil {
		logger = slog.New(baseTextHandler)
	} else {
		baseJsonHandler := slog.NewJSONHandler(jsonWriter, &slog.HandlerOptions{
			Level: logLevel,
		})
		stackJsonHandler := stacktrace.New(baseJsonHandler, &stacktrace.Options{
			Mode: stacktrace.ModePrint,
		})

		multiHandler := multiHandler{stackTextHandler, stackJsonHandler}

		logger = slog.New(multiHandler)
	}
	slog.SetDefault(logger)
}

// LoggerKey key value to store logger in context
const LoggerKey string = "logger"

func GetSlogLoggerFromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(LoggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// multiHandler is a slog.Handler that fans out log records to multiple handlers.
// This implementation is based on community best practices and the proposed
// standard library MultiHandler design [citation:5].
type multiHandler []slog.Handler

// Enabled reports whether any of the child handlers are enabled for the given level.
func (m multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle processes the log record through all enabled child handlers.
// It clones the record for each handler to prevent any mutations from affecting
// subsequent handlers [citation:5].
func (m multiHandler) Handle(ctx context.Context, r slog.Record) error {
	var errs []error
	for _, h := range m {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r.Clone()); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

// WithAttrs returns a new multiHandler with the attributes added to all child handlers.
func (m multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, 0, len(m))
	for _, h := range m {
		newHandlers = append(newHandlers, h.WithAttrs(attrs))
	}
	return multiHandler(newHandlers)
}

// WithGroup returns a new multiHandler with the group name added to all child handlers.
func (m multiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, 0, len(m))
	for _, h := range m {
		newHandlers = append(newHandlers, h.WithGroup(name))
	}
	return multiHandler(newHandlers)
}

// AutoFlushWriter buffers writes and flushes periodically or on demand
type AutoFlushWriter struct {
	writer *bufio.Writer
	file   *os.File
	mu     sync.Mutex
	ticker *time.Ticker
	done   chan bool
}

func NewAutoFlushWriter(file *os.File, bufferSize int, flushInterval time.Duration) *AutoFlushWriter {
	w := &AutoFlushWriter{
		writer: bufio.NewWriterSize(file, bufferSize),
		file:   file,
		done:   make(chan bool),
	}

	// Start auto-flush goroutine
	if flushInterval > 0 {
		w.ticker = time.NewTicker(flushInterval)
		go w.autoFlush()
	}

	return w
}

func (w *AutoFlushWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writer.Write(p)
}

func (w *AutoFlushWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.writer.Flush(); err != nil {
		return err
	}
	return w.file.Sync()
}

func (w *AutoFlushWriter) autoFlush() {
	for {
		select {
		case <-w.ticker.C:
			w.Flush()
		case <-w.done:
			return
		}
	}
}

func (w *AutoFlushWriter) Close() error {
	if w.ticker != nil {
		w.ticker.Stop()
		w.done <- true
	}
	return w.Flush()
}
