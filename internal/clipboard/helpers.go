package clipboard

import (
	"os"
	"regexp"
	"strings"

	"github.com/zeebo/blake3"
)

func trimSpace(value string) string {
	return strings.TrimSpace(value)
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func hashBytes(data []byte) string {
	sum := blake3.Sum256(data)
	return hexString(sum[:])
}

func hexString(data []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, len(data)*2)
	for index, value := range data {
		out[index*2] = digits[value>>4]
		out[index*2+1] = digits[value&0x0f]
	}
	return string(out)
}

var (
	urlPattern   = regexp.MustCompile(`(?i)^https?://[^\s]+$`)
	emailPattern = regexp.MustCompile(`(?i)^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	colorPattern = regexp.MustCompile(`(?i)^(#[0-9a-f]{3,8}|rgba?\([^)]*\)|hsla?\([^)]*\))$`)
)

func detectTextSubKind(text string) string {
	trimmed := strings.TrimSpace(text)
	switch {
	case urlPattern.MatchString(trimmed):
		return "url"
	case emailPattern.MatchString(trimmed):
		return "email"
	case colorPattern.MatchString(trimmed):
		return "color"
	case strings.HasPrefix(trimmed, "~/") || strings.HasPrefix(trimmed, "/"):
		return "path"
	default:
		return ""
	}
}

var secretPattern = regexp.MustCompile(`(?i)(api[_-]?key|secret|token|authorization|bearer)\s*[:=]\s*\S{8,}|-----BEGIN [A-Z ]+PRIVATE KEY-----|sk-[A-Za-z0-9]{20,}`)

func containsSecret(text string) bool {
	return secretPattern.MatchString(text)
}
