package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-review-brief-states-the-conclusion-rules.

// R1, R2: every rule the engine enforces on a typed conclusion is stated in the brief
// that asks for one. The table keys are the blockers the attestation preflight raises
// (by the words that identify them) and the values are the words the brief must carry,
// so a rule added to the preflight without a sentence in the brief fails here.
func TestTheBriefStatesEveryRuleThePreflightEnforces(t *testing.T) {
	store, bundle := briefFixture(t)
	brief, err := store.RenderReviewBrief(bundle.BundleID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	rules := map[string][]string{
		"is a judgment criterion passed with no conclusion":          {"`passed` criterion of kind judgment carries a `rationale`"},
		"cites evidence absent from the sealed bundle":               {"exactly one of the refs listed under Sealed evidence", "never a file and line"},
		"is passed with no evidence":                                 {"exactly one of the refs listed under Sealed evidence"},
		"is disposed as a finding and names none":                    {"disposed `finding` carries as its `evidence` the `id` of a finding"},
		"names finding X, which this attestation does not record":    {"the `id` of a finding listed in `findings`"},
		"passes without answering for <structural fact>":             {"structural fact listed above is answered by the criterion that asks for structural mapping", "`mappings`"},
		"is not-applicable while the bundle seals evidence of class": {"refused when the bundle seals evidence of a class the criterion asks for"},
		"missing criterion":                                          {"every required criterion and every tool above is answered"},
		"review tool has invalid disposition":                        {"`passed`, `failed`, `not-used` or `deferred`, never anything else"},
		"required review tool did not pass":                          {"a tool listed as required must be run and `passed`"},
		"required review tool was not used":                          {"`not-used` is accepted for a required tool only when the plan says no producer exists"},
		"recommended review tool lacks not-used rationale":           {"a recommended tool you did not run is `not-used` with a `rationale`"},
		"finding F is open":                                          {"a draft that approves leaves no finding open", "a finding without them is open and the approval is refused"},
		"invalid review evidence ref":                                {"`command:` followed by the command you ran; a bare command line is refused"},
		"review tool lacks a required evidence class":                {"when the tool lists evidence classes, a ref of one of them that the bundle sealed"},
	}
	for blocker, phrases := range rules {
		for _, phrase := range phrases {
			if !strings.Contains(brief.Text, phrase) {
				t.Errorf("the brief does not state the rule behind %q (%q)", blocker, phrase)
			}
		}
	}
}

// R3: a run briefed with review-brief/3 still renders as it did, without the rules, so its
// digest still verifies; the current template is /4.
func TestAnEarlierConclusionBriefStillRendersWithoutTheRules(t *testing.T) {
	store, bundle := briefFixture(t)
	current, err := store.RenderReviewBrief(bundle.BundleID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	earlier, err := store.renderReviewBriefAs(bundle.BundleID, ReviewBriefKindReview, "", reviewBriefTemplateTypedConclusion)
	if err != nil {
		t.Fatal(err)
	}
	if current.TemplateVersion != "review-brief/6" || earlier.TemplateVersion != "review-brief/3" {
		t.Fatalf("templates: %s %s", current.TemplateVersion, earlier.TemplateVersion)
	}
	if strings.Contains(earlier.Text, "POSE checks the conclusion before it records it") {
		t.Fatal("review-brief/3 changed under the runs briefed with it")
	}
	if !strings.Contains(earlier.Text, "the conclusion POSE records as the attestation") || earlier.Digest == current.Digest {
		t.Fatal("review-brief/3 no longer asks for the conclusion, or has the digest of /4")
	}
	if !reviewBriefAsksConclusion("review-brief/3") || !reviewBriefAsksConclusion("review-brief/4") || !reviewBriefAsksConclusion("review-brief/5") || !reviewBriefAsksConclusion("review-brief/6") || reviewBriefAsksConclusion("review-brief/2") {
		t.Fatal("the templates that ask for a conclusion are /3 and /4")
	}
}

// R4: the Claude reviewer profile allows read-only commands in the three commands it is
// launched with, and nothing that writes.
func TestTheClaudeReviewerProfileAllowsOnlyReadOnlyCommands(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".pose", "policy", "delegation.json"))
	if err != nil {
		t.Skipf("no delegation policy at the repository root: %v", err)
	}
	var policy struct {
		Agents map[string]struct {
			Commands map[string][]string `json:"commands"`
			Resume   struct {
				Command  []string            `json:"command"`
				Commands map[string][]string `json:"commands"`
			} `json:"resume"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	agent, ok := policy.Agents["claude-opus-5-5"]
	if !ok {
		t.Skip("no Claude reviewer profile")
	}
	commands := [][]string{agent.Commands["read"], agent.Resume.Command, agent.Resume.Commands["read"]}
	writers := []string{"assess", "validate", "--seal", "attest", "record", "apply", "close ", "start", "index", "git commit", "git push", "git add", "git checkout", "git reset", "go build", "go install", "rm ", "touch"}
	for i, command := range commands {
		var allowed string
		for j, arg := range command {
			if arg == "--allowedTools" && j+1 < len(command) {
				allowed = command[j+1]
			}
		}
		if allowed == "" {
			t.Fatalf("command %d of the Claude reviewer allows no command: it cannot reproduce what it judges", i)
		}
		for _, want := range []string{"Bash(git log:*)", "Bash(go test:*)", "Bash(pose artifact-check:*)"} {
			if !strings.Contains(allowed, want) {
				t.Errorf("command %d does not allow %s", i, want)
			}
		}
		for _, verb := range writers {
			if strings.Contains(allowed, verb) {
				t.Errorf("command %d allows %q, a verb that writes", i, verb)
			}
		}
	}
}

// A round after the first tells the reviewer how to name the findings it accounts for: the
// preflight refuses a draft whose prior_findings use a bare identifier ("prior finding id F1
// is not a unique carried identifier"), and only the brief of a later round can say so.
func TestTheBriefOfALaterRoundStatesTheIdentifiersOfPriorFindings(t *testing.T) {
	text := renderReviewBriefBody(fixedBriefBundle(), ReviewBriefKindReview, reviewBriefTemplates[ReviewBriefKindReview], "", nil, &ReviewRound{Number: 2}, true, nil)
	for _, phrase := range []string{"the identifier listed under Findings to account for, which carries its round, for example R2-F1, never a bare F1", "a finding you report now takes an `id` that no earlier round used"} {
		if !strings.Contains(text, phrase) {
			t.Errorf("the brief of round 2 does not state the rule behind \"prior finding id is not a unique carried identifier\": %q", phrase)
		}
	}
	earlier := renderReviewBriefBody(fixedBriefBundle(), ReviewBriefKindReview, reviewBriefTemplateConclusions, "", nil, &ReviewRound{Number: 2}, true, nil)
	if strings.Contains(earlier, "never a bare F1") {
		t.Error("review-brief/4 changed under its version: the rule belongs to review-brief/5 and later")
	}
}
