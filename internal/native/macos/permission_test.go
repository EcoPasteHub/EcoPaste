//go:build darwin

package macos

import (
	"errors"
	"testing"
)

func TestFullDiskAccessGranted(t *testing.T) {
	cases := []struct {
		name    string
		results []error
		want    bool
	}{
		{name: "first probe succeeds", results: []error{nil}, want: true},
		{name: "second probe succeeds", results: []error{errors.New("denied"), nil}, want: true},
		{name: "all probes fail", results: []error{errors.New("denied"), errors.New("missing")}, want: false},
		{name: "no paths", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			index := 0
			paths := make([]string, len(tc.results))
			for i := range paths {
				paths[i] = "probe"
			}
			got := fullDiskAccessGranted(paths, func(string) error {
				result := tc.results[index]
				index++
				return result
			})
			if got != tc.want {
				t.Errorf("fullDiskAccessGranted() = %t, want %t", got, tc.want)
			}
		})
	}
}
