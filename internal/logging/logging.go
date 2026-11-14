package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	logFileName = "ecopaste.log"
	maxSizeMB   = 5
	maxBackups  = 3
	maxAgeDays  = 14
)

// Logger sends the standard logger to stderr and a rotating file in the
// data root, masking credentials before they reach either.
type Logger struct {
	file     *lumberjack.Logger
	previous io.Writer
	flags    int
}

// Open starts file logging under dir. It keeps stderr for development and
// records the previous logger so Close restores it.
func Open(dir string) (*Logger, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建日志目录: %w", err)
	}
	file := &lumberjack.Logger{
		Filename:   filepath.Join(dir, logFileName),
		MaxSize:    maxSizeMB,
		MaxBackups: maxBackups,
		MaxAge:     maxAgeDays,
		Compress:   true,
	}
	logger := &Logger{
		file:     file,
		previous: log.Writer(),
		flags:    log.Flags(),
	}
	log.SetOutput(io.MultiWriter(os.Stderr, redactingWriter{dst: file}))
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	return logger, nil
}

// Close stops file logging and restores the logger Open replaced.
func (l *Logger) Close() error {
	log.SetOutput(l.previous)
	log.SetFlags(l.flags)
	return l.file.Close()
}

type redactingWriter struct {
	dst io.Writer
}

func (w redactingWriter) Write(p []byte) (int, error) {
	if _, err := io.WriteString(w.dst, Redact(string(p))); err != nil {
		return 0, err
	}
	return len(p), nil
}

var (
	privateKeyPattern = regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?-----END [A-Z0-9 ]*PRIVATE KEY-----`)
	labeledSecret     = regexp.MustCompile(`(?i)\b((?:api[_-]?key|secret[_-]?key|client[_-]?secret|access[_-]?token|refresh[_-]?token|auth[_-]?token|password|passwd|bearer)\b['"]?\s*[:=]\s*['"]?)([A-Za-z0-9][A-Za-z0-9._~+/=-]{7,})`)
	tokenValue        = regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9_]{36,}|github_pat_[A-Za-z0-9_]{40,}|sk-[A-Za-z0-9_-]{20,}|AKIA[0-9A-Z]{16}|eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,})\b`)
)

// Redact masks credentials that should never reach a log file. It is a
// last line of defense; callers still avoid logging clipboard content.
func Redact(text string) string {
	text = privateKeyPattern.ReplaceAllString(text, "[private key]")
	text = labeledSecret.ReplaceAllString(text, `${1}***`)
	return tokenValue.ReplaceAllString(text, "***")
}
