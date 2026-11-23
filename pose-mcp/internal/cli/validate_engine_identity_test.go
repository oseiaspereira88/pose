package cli

import (
	"runtime/debug"
	"testing"

	"github.com/harne8/pose-mcp/internal/version"
)

// A modified build is not a comparable engine identity (spec
// pose-validate-carries-unchanged-checks, review round 2, N4).
func TestEngineIdentityRefusesModifiedBuilds(t *testing.T) {
	info := func(settings ...debug.BuildSetting) *debug.BuildInfo { return &debug.BuildInfo{Settings: settings} }
	cases := []struct {
		name       string
		info       *debug.BuildInfo
		want       string
		verifiable bool
	}{
		{"clean", info(debug.BuildSetting{Key: "vcs.revision", Value: "abc"}, debug.BuildSetting{Key: "vcs.modified", Value: "false"}), version.Version + "+abc", true},
		{"modified", info(debug.BuildSetting{Key: "vcs.revision", Value: "abc"}, debug.BuildSetting{Key: "vcs.modified", Value: "true"}), version.Version + "+abc.dirty", false},
		{"unstamped", info(), version.Version, true},
	}
	for _, tc := range cases {
		got, verifiable := engineIdentityFrom(tc.info, true)
		if got != tc.want || verifiable != tc.verifiable {
			t.Fatalf("%s: identity %q verifiable %v, want %q %v", tc.name, got, verifiable, tc.want, tc.verifiable)
		}
	}
}
