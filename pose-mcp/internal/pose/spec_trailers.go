package pose

import (
	"bytes"
	"os/exec"
	"strings"
)

// Spec pose-spec-facts-evidence-binding: the facts and the attribution of change sets read the
// trailers of the history in one way.

// SpecTrailerLogArgs are the arguments of the `git log` whose output ParseSpecTrailerLog reads:
// every ref, oldest first, the commits that mention a POSE-Spec trailer, hash and full body.
var SpecTrailerLogArgs = []string{"log", "--all", "--reverse", "--grep=POSE-Spec:", "--format=%H%x00%B%x00"}

// ParseSpecTrailerLog maps each spec slug to the commits that carry its trailer. A trailer is a
// line of the body that starts with `POSE-Spec:` after trimming; the slug is the rest of the
// line, trimmed. It does not use Git's trailer parser.
func ParseSpecTrailerLog(out []byte) map[string][]string {
	parts := bytes.Split(out, []byte{0})
	specCommits := map[string][]string{}
	for i := 0; i+1 < len(parts); i += 2 {
		sha, body := strings.TrimSpace(string(parts[i])), string(parts[i+1])
		for _, line := range strings.Split(body, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "POSE-Spec:") {
				spec := strings.TrimSpace(strings.TrimPrefix(line, "POSE-Spec:"))
				if spec != "" {
					specCommits[spec] = append(specCommits[spec], sha)
				}
			}
		}
	}
	return specCommits
}

// SpecTrailerCommits reads the history of root and returns the commits that carry the trailer of
// slug, the way the attribution of change sets does. An error is a history that could not be
// read, which is not a history without commits.
func SpecTrailerCommits(root, slug string) ([]string, error) {
	out, err := exec.Command("git", append([]string{"-C", root}, SpecTrailerLogArgs...)...).Output()
	if err != nil {
		return nil, err
	}
	return ParseSpecTrailerLog(out)[slug], nil
}
