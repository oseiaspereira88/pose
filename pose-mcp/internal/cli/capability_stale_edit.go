package cli

// Surgical edits to .pose/capabilities/assessment.md's `- stale: ...`
// bullets (spec pose-capability-assessment-triggers R2/R4). Never a full
// re-serialize from the parsed struct: prose commentary below a
// mechanism's bullets is never parsed (internal/pose/capabilities.go) and
// must survive byte-for-byte, exactly like project-state's curated
// sections and markRefreshPending's frontmatter-only edit.

import (
	"fmt"
	"strings"

	"github.com/harne8/pose-mcp/internal/pose"
)

const staleBulletPrefix = "- stale: "

// mechanismBulletRange finds the [headingIdx, bulletEnd) line range for
// one mechanism's flat-bullet block: headingIdx is the "## Mechanism: id"
// line; bulletEnd is the first line after it that is not itself a "- "
// bullet (blank line into prose, the next heading, or EOF).
func mechanismBulletRange(lines []string, id string) (headingIdx, bulletEnd int, found bool) {
	heading := "## Mechanism: " + id
	for i, line := range lines {
		if strings.TrimSpace(line) != heading {
			continue
		}
		j := i + 1
		for j < len(lines) && strings.HasPrefix(lines[j], "- ") {
			j++
		}
		return i, j, true
	}
	return 0, 0, false
}

// addStaleMark inserts (or, for a repeated trigger, replaces in place —
// R7 dedup) a `- stale: ...` bullet in mechanism id's block. Returns the
// updated content; the caller writes it atomically.
func addStaleMark(content, mechanismID string, trigger pose.StaleTrigger) (string, error) {
	lines := strings.Split(content, "\n")
	headingIdx, bulletEnd, found := mechanismBulletRange(lines, mechanismID)
	if !found {
		return "", fmt.Errorf("mechanism %q not found in assessment", mechanismID)
	}
	newLine := staleBulletPrefix + pose.FormatStaleTriggerBullet(trigger)
	for i := headingIdx + 1; i < bulletEnd; i++ {
		if !strings.HasPrefix(lines[i], staleBulletPrefix) {
			continue
		}
		existing, err := pose.ParseStaleTriggerBullet(strings.TrimPrefix(lines[i], staleBulletPrefix))
		if err == nil && existing.Trigger == trigger.Trigger {
			lines[i] = newLine
			return strings.Join(lines, "\n"), nil
		}
	}
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:bulletEnd]...)
	out = append(out, newLine)
	out = append(out, lines[bulletEnd:]...)
	return strings.Join(out, "\n"), nil
}

// clearStaleMarks removes every `- stale: ...` bullet from mechanism id's
// block (R4: `pose assess snapshot` closes the loop) and reports whether
// anything was actually removed, so the caller only links a snapshot to
// mechanisms that genuinely had a pending demand.
func clearStaleMarks(content, mechanismID string) (updated string, cleared bool, err error) {
	lines := strings.Split(content, "\n")
	headingIdx, bulletEnd, found := mechanismBulletRange(lines, mechanismID)
	if !found {
		return content, false, fmt.Errorf("mechanism %q not found in assessment", mechanismID)
	}
	out := make([]string, 0, len(lines))
	out = append(out, lines[:headingIdx+1]...)
	for i := headingIdx + 1; i < bulletEnd; i++ {
		if strings.HasPrefix(lines[i], staleBulletPrefix) {
			cleared = true
			continue
		}
		out = append(out, lines[i])
	}
	out = append(out, lines[bulletEnd:]...)
	return strings.Join(out, "\n"), cleared, nil
}
