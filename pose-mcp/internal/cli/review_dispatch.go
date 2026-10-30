package cli

// pose review dispatch, resume and status were the aliases spec
// pose-delegation-surface deprecated in favour of pose delegate; spec
// pose-delegation-parity-gate removed them (R6, D2). What is left is the
// answer they give now: each names its replacement and runs nothing.

import "io"

// reviewAliasReplacements are the pose delegate commands that replace the
// removed aliases.
var reviewAliasReplacements = map[string]string{
	"dispatch": "pose delegate review|adjudicate|smoke <scope|bundle-id>",
	"resume":   "pose delegate resume <run>",
	"status":   "pose delegate ls",
}

// cmdReviewRemovedAlias answers a removed alias with a usage error naming
// its replacement.
func cmdReviewRemovedAlias(alias string, stderr io.Writer) int {
	replacement := reviewAliasReplacements[alias]
	return usageError(stderr, tr("pose review "+alias+" was removed; use "+replacement, "pose review "+alias+" foi removido; use "+replacement))
}
