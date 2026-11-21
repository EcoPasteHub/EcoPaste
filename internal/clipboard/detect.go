package clipboard

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
)

var (
	urlPattern = regexp.MustCompile(`(?i)^(https?|ftp|file)://[^\s]+$|^www\.[^\s]+\.[^\s]+$`)

	emailPattern = regexp.MustCompile(`^[A-Za-z0-9._%+\-\p{Han}]+@[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)+$`)

	hexColorPattern = regexp.MustCompile(`(?i)^#([0-9a-f]{3}|[0-9a-f]{4}|[0-9a-f]{6}|[0-9a-f]{8})$`)

	colorFunctionPattern = regexp.MustCompile(`(?i)^(rgba?|hsla?|hwb|lab|lch|oklab|oklch|color|color-mix)\(.+\)$`)

	gradientPattern = regexp.MustCompile(`(?i)^(repeating-)?(linear|radial|conic)-gradient\(.+\)$`)
)

// DetectTextSubKind classifies a trimmed plain-text value. The order is
// URL, email, color, then existing absolute path.
func DetectTextSubKind(text string) (database.SubKind, bool) {
	value := strings.TrimSpace(text)
	if value == "" {
		return "", false
	}
	switch {
	case urlPattern.MatchString(value):
		return database.SubURL, true
	case emailPattern.MatchString(value):
		return database.SubEmail, true
	case isColor(value):
		return database.SubColor, true
	case isExistingAbsolutePath(value):
		return database.SubPath, true
	default:
		return "", false
	}
}

func isColor(value string) bool {
	if hexColorPattern.MatchString(value) {
		return true
	}
	if !colorFunctionPattern.MatchString(value) && !gradientPattern.MatchString(value) {
		return false
	}
	return isSafeCSSValue(value)
}

func isSafeCSSValue(value string) bool {
	if strings.ContainsAny(value, ";<>") {
		return false
	}
	lower := strings.ToLower(value)
	for _, token := range []string{"url(", "expression(", "javascript:", "@import"} {
		if strings.Contains(lower, token) {
			return false
		}
	}
	depth := 0
	for _, r := range value {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0
}

func isExistingAbsolutePath(value string) bool {
	if !filepath.IsAbs(value) {
		return false
	}
	_, err := os.Stat(value)
	return err == nil
}
