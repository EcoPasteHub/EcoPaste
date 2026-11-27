package clipboard

import (
	"regexp"
	"strings"
)

var secretPatterns = []*regexp.Regexp{
	// PEM / OpenSSH private key blocks.
	regexp.MustCompile(`-----BEGIN (?:[A-Z0-9 ]+ )?PRIVATE KEY-----`),
	// Tokens with a vendor prefix.
	regexp.MustCompile(`(?i)\b(?:gh[pousr]_[A-Za-z0-9_]{36,}|github_pat_[A-Za-z0-9_]{40,}|sk-[A-Za-z0-9_-]{32,}|sk-proj-[A-Za-z0-9_-]{32,}|xox[baprs]-[A-Za-z0-9-]{20,}|(?:api|access|refresh|secret)[_-]?token_[A-Za-z0-9_-]{24,})\b`),
	// AWS access key IDs. Secret access keys have no stable prefix and are
	// too likely to collide with ordinary text.
	regexp.MustCompile(`\b(?:AKIA|ASIA)[A-Z0-9]{16}\b`),
	// Structurally complete JWTs.
	regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b`),
	// Labeled secrets in config, JSON and env files.
	regexp.MustCompile(`(?i)\b(?:api[_-]?key|secret[_-]?key|client[_-]?secret|access[_-]?token|refresh[_-]?token|auth[_-]?token|bearer)\b['"]?\s*[:=]\s*['"]?[A-Za-z0-9][A-Za-z0-9._~+/=-]{23,}['"]?`),
}

// ContainsSecret reports whether text holds a high-confidence credential.
// The rules stay conservative so ordinary prose and long paths do not
// get dropped.
func ContainsSecret(text string) bool {
	value := strings.TrimSpace(text)
	if value == "" {
		return false
	}
	for _, pattern := range secretPatterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	return false
}

// MaskSecret hides the middle of each line while keeping enough edge
// characters for the user to recognize the item.
func MaskSecret(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = maskLine(line)
	}
	return strings.Join(lines, "\n")
}

func maskLine(line string) string {
	const visibleEdge = 4
	const maxMask = 8

	chars := []rune(line)
	if len(chars) == 0 {
		return ""
	}
	if len(chars) <= visibleEdge*2 {
		return strings.Repeat("*", len(chars))
	}
	hidden := min(len(chars)-visibleEdge*2, maxMask)
	return string(chars[:visibleEdge]) + strings.Repeat("*", hidden) + string(chars[len(chars)-visibleEdge:])
}
