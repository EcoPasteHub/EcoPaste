package app

import (
	"testing"

	"github.com/nsgo-dev/nsgo/core"
)

func TestDevelopmentIdentitySeparatesReleaseBuilds(t *testing.T) {
	cases := []struct {
		name    string
		info    core.AppInfo
		wantErr bool
	}{
		{
			name: "development bundle",
			info: core.AppInfo{Development: true, ID: developmentBundleID},
		},
		{
			name:    "release bundle",
			info:    core.AppInfo{Development: true, ID: "com.ayangweb.eco-paste"},
			wantErr: true,
		},
		{
			name: "release build",
			info: core.AppInfo{ID: "com.ayangweb.eco-paste"},
		},
		{
			name: "plain go build without injected identity",
			info: core.AppInfo{Development: true},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := developmentIdentityError(tc.info)
			if (err != nil) != tc.wantErr {
				t.Fatalf("developmentIdentityError(%+v) = %v, want error %t", tc.info, err, tc.wantErr)
			}
		})
	}
}
