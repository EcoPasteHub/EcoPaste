package clipboard

import (
	"strings"
	"testing"
)

func TestContainsSecret(t *testing.T) {
	tests := map[string]bool{
		"github token":       true,
		"github pat":         true,
		"openai key":         true,
		"openai project key": true,
		"slack token":        true,
		"api token":          true,
		"aws key":            true,
		"jwt":                true,
		"private key":        true,
		"json api key":       true,
		"single quoted key":  true,
		"ordinary chinese":   false,
		"verification code":  false,
		"url":                false,
		"short aws prefix":   false,
		"empty":              false,
	}
	values := map[string]string{
		"github token":       "ghp_" + strings.Repeat("a", 36),
		"github pat":         "github_pat_" + strings.Repeat("b", 40),
		"openai key":         "sk-" + strings.Repeat("c", 32),
		"openai project key": "sk-proj-" + strings.Repeat("d", 32),
		"slack token":        "xoxb-" + strings.Repeat("e", 20),
		"api token":          "api_token_" + strings.Repeat("f", 24),
		"aws key":            "AKIA" + strings.Repeat("G", 16),
		"jwt":                "eyJ" + strings.Repeat("h", 8) + "." + strings.Repeat("i", 8) + "." + strings.Repeat("j", 8),
		"private key":        "-----BEGIN OPENSSH PRIVATE KEY-----\nabc",
		"json api key":       `{"api_key": "` + strings.Repeat("i", 24) + `"}`,
		"single quoted key":  `'access_token': '` + strings.Repeat("j", 24) + `'`,
		"ordinary chinese":   "这是一段普通文本，只是提到了 token 这个词。",
		"verification code":  "验证码 123456",
		"url":                "https://example.com/path/to/resource",
		"short aws prefix":   "AKIA is just a word",
		"empty":              "   ",
	}
	for name, want := range tests {
		if got := ContainsSecret(values[name]); got != want {
			t.Errorf("%s: ContainsSecret = %v, want %v", name, got, want)
		}
	}
}

func TestMaskSecret(t *testing.T) {
	tests := []struct{ input, want string }{
		{"", ""},
		{"short", "*****"},
		{"12345678", "********"},
		{"abcdefghijkl", "abcd****ijkl"},
		{"abcdefghijklmnopqrstuvwxyz", "abcd********wxyz"},
		{"abcdefghijkl\nshort", "abcd****ijkl\n*****"},
	}
	for _, tt := range tests {
		if got := MaskSecret(tt.input); got != tt.want {
			t.Errorf("MaskSecret(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
