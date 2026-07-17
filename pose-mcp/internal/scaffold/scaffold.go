// Package scaffold embeds the POSE distribution so the unified binary can
// install POSE without a repository clone (spec pose-cli-embed-standalone).
//
// The dist/ tree is GENERATED from pose-dist/ by gen/main.go — do not edit
// by hand; run `go generate ./internal/scaffold` after changing pose-dist/.
// scaffold_test.go fails the build's test gate on any drift.
package scaffold

//go:generate go run ./gen

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// Dist returns the embedded POSE distribution rooted at its top level.
func Dist() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err) // impossible: dist is embedded at compile time
	}
	return sub
}

// ClaudeSkillLinks lists the .claude/skills symlinks to recreate at install
// time (go:embed cannot carry symlinks). Name → relative target.
var ClaudeSkillLinks = map[string]string{
	"pose-adr":                   "../../.agents/skills/pose-adr",
	"pose-bugfix":                "../../.agents/skills/pose-bugfix",
	"pose-doc-update":            "../../.agents/skills/pose-doc-update",
	"pose-feature":               "../../.agents/skills/pose-feature",
	"pose-knowledge":             "../../.agents/skills/pose-knowledge",
	"pose-recurrence-escalation": "../../.agents/skills/pose-recurrence-escalation",
	"pose-review":                "../../.agents/skills/pose-review",
	"pose-spec-closeout":         "../../.agents/skills/pose-spec-closeout",
	"pose-test-plan":             "../../.agents/skills/pose-test-plan",
}
