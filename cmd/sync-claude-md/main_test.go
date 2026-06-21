package main

import (
	"runtime/debug"
	"testing"
)

// TestVersionFromBuildInfo covers the two independent sources build info can
// supply — module version and VCS stamps — including the cases where one or
// both are absent (e.g. `go install pkg@latest`, which resolves a module
// version but has no VCS checkout to stamp from).
func TestVersionFromBuildInfo(t *testing.T) {
	cases := []struct {
		name        string
		info        *debug.BuildInfo
		wantVersion string
		wantCommit  string
		wantDate    string
	}{
		{
			name: "module version and vcs stamps present",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "v1.2.3"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc123"},
					{Key: "vcs.time", Value: "2024-01-01T00:00:00Z"},
				},
			},
			wantVersion: "v1.2.3",
			wantCommit:  "abc123",
			wantDate:    "2024-01-01T00:00:00Z",
		},
		{
			name: "module version only, no vcs stamps (go install pkg@latest)",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "v1.2.3"},
			},
			wantVersion: "v1.2.3",
			wantCommit:  "none",
			wantDate:    "unknown",
		},
		{
			name: "devel version is not a usable replacement",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "(devel)"},
			},
			wantVersion: "dev",
			wantCommit:  "none",
			wantDate:    "unknown",
		},
		{
			name:        "empty build info changes nothing",
			info:        &debug.BuildInfo{},
			wantVersion: "dev",
			wantCommit:  "none",
			wantDate:    "unknown",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotVersion, gotCommit, gotDate := versionFromBuildInfo(tc.info, "dev", "none", "unknown")
			if gotVersion != tc.wantVersion {
				t.Errorf("version = %q, want %q", gotVersion, tc.wantVersion)
			}
			if gotCommit != tc.wantCommit {
				t.Errorf("commit = %q, want %q", gotCommit, tc.wantCommit)
			}
			if gotDate != tc.wantDate {
				t.Errorf("date = %q, want %q", gotDate, tc.wantDate)
			}
		})
	}
}
