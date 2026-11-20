package logging

import (
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	logFileName     = "ecopaste.log"
	crashFileName   = "ecopaste-crash.log"
	crashFilePrefix = "ecopaste-crash-"
	maxSizeMB       = 5
	maxQueueEntries = 512
	logRetention    = 14 * 24 * time.Hour
)

// Level is the minimum severity written by the application's structured logger.
type Level string

const (
	LevelAll   Level = "all"
	LevelError Level = "error"
	LevelWarn  Level = "warn"
	LevelInfo  Level = "info"
	LevelDebug Level = "debug"
)

// Logger owns the application's bounded asynchronous log pipeline.
type Logger struct {
	level             *slog.LevelVar
	rotating          *rotatingWriter
	async             *asyncWriter
	previous          *slog.Logger
	previousOut       io.Writer
	previousLogFlags  int
	previousLogPrefix string
	crashOutput       bool
	crashPath         string
	closed            bool
	mu                sync.Mutex
}

var activeLogger struct {
	sync.Mutex
	active *Logger
}

// Open starts structured, redacted file logging in dir. The standard log
// package is bridged into the same pipeline for dependencies that still use it.
func Open(dir string) (*Logger, error) {
	activeLogger.Lock()
	if activeLogger.active != nil {
		activeLogger.Unlock()
		return nil, errors.New("application logger already open")
	}
	activeLogger.active = &Logger{}
	activeLogger.Unlock()

	logger, err := open(dir)
	if err != nil {
		activeLogger.Lock()
		activeLogger.active = nil
		activeLogger.Unlock()
		return nil, err
	}
	activeLogger.Lock()
	activeLogger.active = logger
	activeLogger.Unlock()
	return logger, nil
}

// Configure updates the active logger when settings change outside the
// preferences view, such as during a backup merge.
func Configure(level Level) error {
	activeLogger.Lock()
	logger := activeLogger.active
	activeLogger.Unlock()
	if logger == nil || logger.level == nil {
		return nil
	}
	return logger.SetLevel(level)
}

func open(dir string) (*Logger, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	rotating := &rotatingWriter{dir: dir}
	if err := rotating.configure(); err != nil {
		return nil, err
	}
	level := &slog.LevelVar{}
	level.Set(slog.LevelError)
	output := &redactingWriter{dst: bestEffortTee{writers: []io.Writer{os.Stderr, rotating}}}
	async := newAsyncWriter(maxQueueEntries, output)
	logger := &Logger{
		level:             level,
		rotating:          rotating,
		async:             async,
		previous:          slog.Default(),
		previousOut:       log.Writer(),
		previousLogFlags:  log.Flags(),
		previousLogPrefix: log.Prefix(),
	}
	logger.configureCrashOutput(dir)
	if err := rotating.cleanup(); err != nil {
		slog.Error("remove expired log files", "module", "logging", "error", err)
	}
	logger.rotating.startCleanup()
	handler := slog.NewJSONHandler(async, &slog.HandlerOptions{Level: level, AddSource: true})
	slog.SetDefault(slog.New(handler))
	log.SetOutput(standardLogWriter{logger: slog.Default()})
	log.SetFlags(log.Lshortfile)
	log.SetPrefix("")
	return logger, nil
}

// SetLevel changes the minimum severity immediately.
func (l *Logger) SetLevel(level Level) error {
	if l == nil {
		return errors.New("logger is not available")
	}
	value, err := parseLevel(level)
	if err != nil {
		return err
	}
	l.level.Set(value)
	return nil
}

func parseLevel(level Level) (slog.Level, error) {
	switch level {
	case LevelAll:
		return slog.Level(math.MinInt), nil
	case LevelError:
		return slog.LevelError, nil
	case LevelWarn:
		return slog.LevelWarn, nil
	case LevelInfo:
		return slog.LevelInfo, nil
	case LevelDebug:
		return slog.LevelDebug, nil
	default:
		return 0, fmt.Errorf("invalid log level %q", level)
	}
}

// Directory returns the active log directory.
func (l *Logger) Directory() string {
	if l == nil || l.rotating == nil {
		return ""
	}
	return l.rotating.dir
}

// Close flushes queued records and restores the process's previous loggers.
func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	l.mu.Unlock()

	if l.crashOutput {
		_ = setCrashOutput(nil)
	}
	if info, err := os.Stat(l.crashPath); err == nil && info.Size() == 0 {
		_ = os.Remove(l.crashPath)
	}
	if l.previous != nil {
		slog.SetDefault(l.previous)
	}
	log.SetOutput(l.previousOut)
	log.SetFlags(l.previousLogFlags)
	log.SetPrefix(l.previousLogPrefix)
	err := l.async.close()
	if closeErr := l.rotating.Close(); err == nil {
		err = closeErr
	}
	activeLogger.Lock()
	activeLogger.active = nil
	activeLogger.Unlock()
	return err
}

type standardLogWriter struct {
	logger *slog.Logger
}

func (w standardLogWriter) Write(data []byte) (int, error) {
	message := strings.TrimSpace(string(data))
	if message != "" {
		caller := ""
		if separator := strings.Index(message, ": "); separator >= 0 {
			caller = message[:separator]
			message = message[separator+2:]
		}
		w.logger.Error(message, "module", "standard", "caller", caller)
	}
	return len(data), nil
}

type rotatingWriter struct {
	dir    string
	mu     sync.Mutex
	file   *lumberjack.Logger
	stop   chan struct{}
	done   chan struct{}
	closed bool
}

func (w *rotatingWriter) configure() error {
	if err := os.MkdirAll(w.dir, 0o700); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}
	_ = os.Chmod(w.dir, 0o700)
	w.file = newRotator(w.dir)
	return nil
}

func (w *rotatingWriter) startCleanup() {
	w.mu.Lock()
	if w.closed || w.stop != nil {
		w.mu.Unlock()
		return
	}
	w.stop, w.done = make(chan struct{}), make(chan struct{})
	stop, done := w.stop, w.done
	w.mu.Unlock()
	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := w.cleanup(); err != nil {
					slog.Error("remove expired log files", "module", "logging", "error", err)
				}
			case <-stop:
				return
			}
		}
	}()
}

func (w *rotatingWriter) cleanup() error {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return fmt.Errorf("read log directory: %w", err)
	}
	cutoff := time.Now().Add(-logRetention)
	var backups []os.FileInfo
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !isRotatedLog(name) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(w.dir, name))
			continue
		}
		backups = append(backups, info)
	}
	if len(backups) <= 20 {
		return nil
	}
	sort.Slice(backups, func(i, j int) bool { return backups[i].ModTime().After(backups[j].ModTime()) })
	var firstErr error
	for _, info := range backups[20:] {
		if err := os.Remove(filepath.Join(w.dir, info.Name())); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func isRotatedLog(name string) bool {
	if !strings.HasPrefix(name, "ecopaste-") {
		return false
	}
	return strings.HasSuffix(name, ".log") || strings.HasSuffix(name, ".log.gz")
}

func (w *rotatingWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || w.file == nil {
		return len(data), nil
	}
	n, err := w.file.Write(data)
	if err != nil {
		return n, err
	}
	return n, nil
}

func (w *rotatingWriter) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	stop, done := w.stop, w.done
	if w.file == nil {
		w.mu.Unlock()
		return nil
	}
	file := w.file
	w.mu.Unlock()
	if stop != nil {
		close(stop)
		<-done
	}
	return file.Close()
}

func newRotator(dir string) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   filepath.Join(dir, logFileName),
		MaxSize:    maxSizeMB,
		MaxBackups: 20,
		MaxAge:     0,
		Compress:   true,
	}
}

type asyncWriter struct {
	queue   chan []byte
	done    chan struct{}
	output  *redactingWriter
	mu      sync.Mutex
	closed  bool
	dropped uint64
}

func newAsyncWriter(capacity int, output *redactingWriter) *asyncWriter {
	w := &asyncWriter{queue: make(chan []byte, capacity), done: make(chan struct{}), output: output}
	go func() {
		for record := range w.queue {
			_, _ = w.output.Write(record)
		}
		close(w.done)
	}()
	return w
}

func (w *asyncWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return len(data), nil
	}
	copyOfData := append([]byte(nil), data...)
	select {
	case w.queue <- copyOfData:
		if w.dropped > 0 {
			warning := []byte(fmt.Sprintf("{\"time\":%q,\"level\":\"WARN\",\"msg\":\"dropped log records\",\"module\":\"logging\",\"count\":%d}\n",
				time.Now().Format(time.RFC3339Nano), w.dropped))
			select {
			case w.queue <- warning:
				w.dropped = 0
			default:
			}
		}
	default:
		w.dropped++
		if strings.Contains(string(data), `"level":"ERROR"`) || strings.Contains(string(data), `"level":"WARN"`) {
			_, _ = w.output.Write(data)
		}
	}
	return len(data), nil
}

func (w *asyncWriter) close() error {
	w.mu.Lock()
	dropped := w.dropped
	w.dropped = 0
	if !w.closed {
		w.closed = true
		close(w.queue)
	}
	done := w.done
	w.mu.Unlock()
	<-done
	if dropped > 0 {
		_, _ = w.output.Write([]byte(fmt.Sprintf("{\"time\":%q,\"level\":\"WARN\",\"msg\":\"dropped log records\",\"count\":%d}\n",
			time.Now().Format(time.RFC3339Nano), dropped)))
	}
	return nil
}

type redactingWriter struct {
	dst io.Writer
}

func (w *redactingWriter) Write(data []byte) (int, error) {
	clean := Redact(string(data))
	_, _ = io.WriteString(w.dst, clean)
	return len(data), nil
}

type bestEffortTee struct {
	writers []io.Writer
}

func (w bestEffortTee) Write(data []byte) (int, error) {
	for _, dst := range w.writers {
		_, _ = dst.Write(data)
	}
	return len(data), nil
}

func (l *Logger) configureCrashOutput(dir string) {
	legacyPath := filepath.Join(dir, crashFileName)
	if info, err := os.Stat(legacyPath); err == nil {
		if info.Size() == 0 {
			_ = os.Remove(legacyPath)
		} else {
			archive := filepath.Join(dir, crashFilePrefix+time.Now().Format("20060102T150405.000000000")+".log")
			if os.Rename(legacyPath, archive) == nil {
				_ = os.Chmod(archive, 0o600)
			}
		}
	}
	path := filepath.Join(dir, crashFilePrefix+time.Now().Format("20060102T150405.000000000")+".log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	_ = file.Chmod(0o600)
	if setCrashOutput(file) == nil {
		l.crashOutput = true
		l.crashPath = path
	}
	_ = file.Close()
}

func setCrashOutput(file *os.File) error {
	return debug.SetCrashOutput(file, debug.CrashOptions{})
}

var (
	privateKeyPattern = regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?-----END [A-Z0-9 ]*PRIVATE KEY-----`)
	labeledSecret     = regexp.MustCompile(`(?i)\b((?:api[_-]?key|secret(?:[_-]?key)?|client[_-]?secret|access[_-]?token|refresh[_-]?token|auth[_-]?token|token|authorization|password|passwd|bearer)\b['"]?\s*[:=]\s*['"]?)([^\s,'";}]+)`)
	urlCredentials    = regexp.MustCompile(`(?i)\b(https?://[^:/\s@]+:)[^@\s/]+@`)
	tokenValue        = regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9_]{36,}|github_pat_[A-Za-z0-9_]{40,}|sk-[A-Za-z0-9_-]{20,}|AKIA[0-9A-Z]{16}|eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,})\b`)
)

// Redact masks credentials that should never reach a log file. Callers must
// still avoid recording clipboard content and other sensitive user data.
func Redact(text string) string {
	text = privateKeyPattern.ReplaceAllString(text, "[private key]")
	text = labeledSecret.ReplaceAllString(text, `${1}***`)
	text = urlCredentials.ReplaceAllString(text, `${1}***@`)
	return tokenValue.ReplaceAllString(text, "***")
}
