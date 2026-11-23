package settings

import (
	"runtime"
	"testing"
)

func TestDefaultSourceAppsHaveNoStoredRules(t *testing.T) {
	if rules := Default().Clipboard.SourceApps.Rules; len(rules) != 0 {
		t.Fatalf("default rules = %+v, want none", rules)
	}
}

func TestDefaultSourceAppModeFollowsPlatform(t *testing.T) {
	for _, id := range []string{"com.apple.keychainaccess", "com.apple.Passwords"} {
		want := SourceAppBlockCapture
		if runtime.GOOS != "darwin" {
			want = SourceAppAllowed
		}
		if got := DefaultSourceAppMode(id); got != want {
			t.Errorf("%s default = %q, want %q", id, got, want)
		}
	}
	if got := DefaultSourceAppMode("com.example.editor"); got != SourceAppAllowed {
		t.Errorf("normal app default = %q", got)
	}
}

func TestSourceAppsOverrideDefaultAndBlockFlags(t *testing.T) {
	apps := SourceApps{Rules: []SourceAppRule{
		{ID: "capture", Mode: SourceAppBlockCapture},
		{ID: "wake", Mode: SourceAppBlockWake},
		{ID: "both", Mode: SourceAppBlockCaptureAndWake},
	}}
	cases := []struct {
		id            string
		blocksCapture bool
		blocksWake    bool
	}{
		{"capture", true, false},
		{"wake", false, true},
		{"both", true, true},
		{"other", false, false},
	}
	for _, tc := range cases {
		if got := apps.BlocksCapture(tc.id); got != tc.blocksCapture {
			t.Errorf("%s blocks capture = %v", tc.id, got)
		}
		if got := apps.BlocksWake(tc.id); got != tc.blocksWake {
			t.Errorf("%s blocks wake = %v", tc.id, got)
		}
	}
}

func TestSourceAppsListedIDsKeepDefaultsAndRules(t *testing.T) {
	ids := SourceApps{Rules: []SourceAppRule{
		{ID: "com.example.app", Mode: SourceAppBlockWake},
		{ID: "com.apple.Passwords", Mode: SourceAppBlockWake},
	}}.ListedIDs()
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			t.Errorf("duplicate listed id %q", id)
		}
		seen[id] = true
	}
	if !seen["com.example.app"] || !seen["com.apple.Passwords"] {
		t.Fatalf("listed ids = %v", ids)
	}
	if runtime.GOOS == "darwin" && !seen["com.apple.keychainaccess"] {
		t.Fatalf("keychain access not listed: %v", ids)
	}
}

func TestSourceAppsCanExplicitlyAllowDefaultBlockedApp(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("password-manager defaults only apply on macOS")
	}
	apps := SourceApps{Rules: []SourceAppRule{{ID: "com.apple.Passwords", Mode: SourceAppAllowed}}}
	if got := apps.ModeFor("com.apple.Passwords"); got != SourceAppAllowed {
		t.Errorf("explicit allowed mode = %q", got)
	}
	if apps.BlocksCapture("com.apple.Passwords") {
		t.Error("explicit allowed app still blocks capture")
	}
	// The override is kept because it differs from the platform default, while
	// restating the default is dropped.
	rules := NormalizeSourceAppRules(apps.Rules)
	if len(rules) != 1 || rules[0].Mode != SourceAppAllowed {
		t.Fatalf("normalized = %+v", rules)
	}
	if got := NormalizeSourceAppRules([]SourceAppRule{{ID: "com.apple.Passwords", Mode: SourceAppBlockCapture}}); len(got) != 0 {
		t.Fatalf("default restatement kept: %+v", got)
	}
}

func TestNormalizeSourceAppRulesKeepsOneValidRule(t *testing.T) {
	rules := []SourceAppRule{
		{ID: "b", Mode: SourceAppBlockWake},
		{ID: "a", Mode: SourceAppBlockCapture},
		{ID: "a", Mode: SourceAppBlockCaptureAndWake},
		{ID: "c", Mode: SourceAppAllowed},
		{ID: "", Mode: SourceAppBlockCapture},
		{ID: "d", Mode: "unknown"},
	}
	got := NormalizeSourceAppRules(rules)
	want := []SourceAppRule{
		{ID: "a", Mode: SourceAppBlockCapture},
		{ID: "b", Mode: SourceAppBlockWake},
	}
	if len(got) != len(want) {
		t.Fatalf("normalized = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("normalized[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}
