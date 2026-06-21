package main

import (
	"runtime/debug"
	"testing"
)

// TestVersionFromBuildInfo covers the two independent sources build info can
// supply — module version and VCS revision — including the cases where one
// or both are absent (e.g. `go install pkg@latest`, which resolves a module
// version but has no VCS checkout to stamp from) and where the module
// version is "(devel)" but a VCS revision is still present (a plain `go
// build` in a local checkout).
func TestVersionFromBuildInfo(t *testing.T) {
	cases := []struct {
		name        string
		info        *debug.BuildInfo
		wantVersion string
		wantCommit  string
	}{
		{
			name: "module version and vcs revision present",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "v1.2.3"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc123"},
					{Key: "vcs.time", Value: "2024-01-01T00:00:00Z"},
				},
			},
			wantVersion: "v1.2.3",
			wantCommit:  "abc123",
		},
		{
			name: "module version only, no vcs stamps (go install pkg@latest)",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "v1.2.3"},
			},
			wantVersion: "v1.2.3",
			wantCommit:  "none",
		},
		{
			name: "devel version is not a usable replacement",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "(devel)"},
			},
			wantVersion: "dev",
			wantCommit:  "none",
		},
		{
			name: "devel version but vcs revision present (local go build)",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "(devel)"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc123"},
					{Key: "vcs.time", Value: "2024-01-01T00:00:00Z"},
				},
			},
			wantVersion: "dev",
			wantCommit:  "abc123",
		},
		{
			name:        "empty build info changes nothing",
			info:        &debug.BuildInfo{},
			wantVersion: "dev",
			wantCommit:  "none",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotVersion, gotCommit := versionFromBuildInfo(tc.info, "dev", "none")
			if gotVersion != tc.wantVersion {
				t.Errorf("version = %q, want %q", gotVersion, tc.wantVersion)
			}
			if gotCommit != tc.wantCommit {
				t.Errorf("commit = %q, want %q", gotCommit, tc.wantCommit)
			}
		})
	}
}
