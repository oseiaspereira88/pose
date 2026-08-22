package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func cmdReviewCheck(root string, args []string, stdout, stderr io.Writer) int {
	ref, jsonOutput, ok := parseScopeCheckArgs("review-check", args, stderr)
	if !ok {
		return 2
	}
	eval, err := (posemodel.Store{Root: root}).ReviewCheck(ref)
	if err != nil {
		fmt.Fprintf(stderr, "pose review-check: %v\n", err)
		return 1
	}
	if jsonOutput {
		if code := writeJSON(stdout, eval); code != 0 {
			return code
		}
		if eval.Required && !eval.Approved {
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "review.scope=%s\nreview.required=%t\nreview.profile=%s\nreview.digest=%s\nreview.fresh=%t\nreview.approved=%t\n", eval.Scope, eval.Required, eval.Profile, eval.ScopeDigest, eval.Fresh, eval.Approved)
	for _, warning := range eval.Warnings {
		fmt.Fprintf(stdout, "[WARN] %s\n", warning)
	}
	for _, blocker := range eval.Blockers {
		fmt.Fprintf(stderr, "[ERROR] %s\n", blocker)
	}
	if eval.Required && !eval.Approved {
		return 1
	}
	return 0
}

func cmdReviewPlan(root string, args []string, stdout, stderr io.Writer) int {
	ref, jsonOutput, explain := "", false, false
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		case "--explain":
			explain = true
		default:
			if strings.HasPrefix(arg, "-") || ref != "" {
				fmt.Fprintln(stderr, "Usage: pose review-plan <spec:slug|milestone:roadmap/id|roadmap:slug> [--json] [--explain]")
				return 2
			}
			ref = arg
		}
	}
	if ref == "" {
		fmt.Fprintln(stderr, "Usage: pose review-plan <spec:slug|milestone:roadmap/id|roadmap:slug> [--json] [--explain]")
		return 2
	}
	plan, err := (posemodel.Store{Root: root}).ReviewPlan(ref)
	if err != nil {
		fmt.Fprintf(stderr, "pose review-plan: %v\n", err)
		return 1
	}
	if jsonOutput {
		if code := writeJSON(stdout, plan); code != 0 {
			return code
		}
		if len(plan.Blockers) > 0 {
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "review_plan.scope=%s\nreview_plan.scope_digest=%s\nreview_plan.plan_digest=%s\nreview_plan.base_profile=%s\nreview_plan.independence=%s\nreview_plan.components=%d\nreview_plan.criteria=%d\nreview_plan.tools=%d\n", plan.Scope, plan.ScopeDigest, plan.PlanDigest, plan.BaseProfile, plan.Independence, len(plan.Components), len(plan.Criteria), len(plan.Tools))
	for _, warning := range groupedReviewPlanWarnings(plan.Warnings) {
		fmt.Fprintf(stdout, "[WARN] %s\n", warning)
	}
	for _, blocker := range plan.Blockers {
		fmt.Fprintf(stderr, "[ERROR] %s\n", blocker)
	}
	if explain {
		for _, event := range plan.Explain {
			fmt.Fprintf(stdout, "explain=%s\n", event)
		}
		for _, profile := range plan.SelectedProfiles {
			fmt.Fprintf(stdout, "profile.%s=order:%d category:%s source:%s components:%s rationale:%s\n", profile.Ref, profile.Order, profile.Category, profile.Source, strings.Join(profile.Components, ","), profile.Rationale)
		}
		for _, criterion := range plan.Criteria {
			fmt.Fprintf(stdout, "criterion.%s=required:%t profiles:%s rules:%s evidence:%s\n", criterion.ID, criterion.Required, strings.Join(criterion.Profiles, ","), strings.Join(criterion.Rules, ","), strings.Join(criterion.EvidenceClasses, ","))
		}
		for _, item := range actionableReviewPlanTools(plan.Tools) {
			tool := item.Tool
			fmt.Fprintf(stdout, "tool.%s.%s=requiredness:%s args:%s preconditions:%s criteria:%s evidence:%s rationale:%s\n", item.Phase, tool.ID, tool.Requiredness, strings.Join(tool.Args, " "), strings.Join(tool.Preconditions, ","), strings.Join(tool.Criteria, ","), strings.Join(tool.EvidenceClasses, ","), tool.Rationale)
		}
	}
	if len(plan.Blockers) > 0 {
		return 1
	}
	return 0
}

func groupedReviewPlanWarnings(warnings []string) []string {
	groups := map[string][]string{}
	plain := []string{}
	for _, warning := range warnings {
		if strings.HasPrefix(warning, "unmapped review component ") {
			groups["unmapped-review-component"] = append(groups["unmapped-review-component"], strings.TrimPrefix(warning, "unmapped review component "))
		} else {
			plain = append(plain, warning)
		}
	}
	result := append([]string{}, plain...)
	for code, values := range groups {
		sort.Strings(values)
		representative := values
		if len(representative) > 3 {
			representative = representative[:3]
		}
		result = append(result, fmt.Sprintf("%s count=%d examples=%s", code, len(values), strings.Join(representative, ",")))
	}
	sort.Strings(result)
	return result
}

type actionableReviewTool struct {
	Phase string
	Tool  posemodel.ReviewPlanTool
}

func actionableReviewPlanTools(tools []posemodel.ReviewPlanTool) []actionableReviewTool {
	result := []actionableReviewTool{}
	seen := map[string]bool{}
	for _, tool := range tools {
		phase := "recommended"
		if containsCLIReviewValue(tool.Preconditions, "review-complete") {
			phase = "completion-deferred"
		} else if tool.Requiredness == "required" {
			phase = "required"
		}
		key := phase + "\x00" + strings.Join(tool.Args, "\x00")
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, actionableReviewTool{Phase: phase, Tool: tool})
	}
	order := map[string]int{"required": 0, "recommended": 1, "completion-deferred": 2}
	sort.SliceStable(result, func(i, j int) bool {
		if order[result[i].Phase] != order[result[j].Phase] {
			return order[result[i].Phase] < order[result[j].Phase]
		}
		return result[i].Tool.ID < result[j].Tool.ID
	})
	return result
}

func cmdCloseoutCheck(root string, args []string, stdout, stderr io.Writer) int {
	ref, jsonOutput, ok := parseScopeCheckArgs("closeout-check", args, stderr)
	if !ok {
		return 2
	}
	state, err := (posemodel.Store{Root: root}).GetCloseoutState(ref)
	if err != nil {
		fmt.Fprintf(stderr, "pose closeout-check: %v\n", err)
		return 1
	}
	if jsonOutput {
		if code := writeJSON(stdout, state); code != 0 {
			return code
		}
		if !state.Terminal {
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "closeout.scope=%s\ncloseout.digest=%s\ncloseout.lifecycle_done=%t\ncloseout.review_approved=%t\ncloseout.terminal=%t\ncloseout.next_action=%s\n", state.Scope, state.ScopeDigest, state.LifecycleDone, state.Review.Approved, state.Terminal, state.NextAction)
	for _, blocker := range state.Blockers {
		fmt.Fprintf(stderr, "[ERROR] %s\n", blocker)
	}
	if !state.Terminal {
		return 1
	}
	return 0
}

func parseScopeCheckArgs(command string, args []string, stderr io.Writer) (string, bool, bool) {
	jsonOutput := false
	ref := ""
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		default:
			if strings.HasPrefix(arg, "-") || ref != "" {
				fmt.Fprintf(stderr, "Usage: pose %s <spec:slug|milestone:roadmap/id|roadmap:slug> [--json]\n", command)
				return "", false, false
			}
			ref = arg
		}
	}
	if ref == "" {
		fmt.Fprintf(stderr, "Usage: pose %s <spec:slug|milestone:roadmap/id|roadmap:slug> [--json]\n", command)
		return "", false, false
	}
	return ref, jsonOutput, true
}

func writeJSON(w io.Writer, value any) int {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return 1
	}
	return 0
}

func cmdReview(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: pose review <bundle|attest|auto-attest|verify|record> ...")
		return 2
	}
	switch args[0] {
	case "bundle":
		return cmdReviewBundle(root, args[1:], stdout, stderr)
	case "attest":
		return cmdReviewAttest(root, args[1:], stdout, stderr)
	case "auto-attest":
		return cmdReviewAutoAttest(root, args[1:], stdout, stderr)
	case "verify":
		return cmdReviewVerify(root, args[1:], stdout, stderr)
	case "record":
		return cmdReviewRecord(root, args[1:], stdout, stderr)
	default:
		fmt.Fprintln(stderr, "Usage: pose review <bundle|attest|auto-attest|verify|record> ...")
		return 2
	}
}

func cmdReviewBundle(root string, args []string, stdout, stderr io.Writer) int {
	ref, jsonOutput, explain, seal := "", false, false, false
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		case "--explain":
			explain = true
		case "--seal":
			seal = true
		default:
			if strings.HasPrefix(arg, "-") || ref != "" {
				fmt.Fprintln(stderr, "Usage: pose review bundle <scope> [--json] [--explain] [--seal]")
				return 2
			}
			ref = arg
		}
	}
	if ref == "" {
		fmt.Fprintln(stderr, "Usage: pose review bundle <scope> [--json] [--explain] [--seal]")
		return 2
	}
	store := posemodel.Store{Root: root}
	var bundle posemodel.ReviewBundle
	var err error
	if seal {
		bundle, err = store.SealReviewBundle(ref, time.Now())
	} else {
		bundle, err = store.PrepareReviewBundle(ref)
	}
	if err != nil {
		fmt.Fprintf(stderr, "pose review bundle: %v\n", err)
		if bundle.BundleID == "" {
			return 1
		}
	}
	if jsonOutput {
		if code := writeJSON(stdout, bundle); code != 0 {
			return code
		}
	} else {
		fmt.Fprintf(stdout, "review_bundle.id=%s\nreview_bundle.digest=%s\nreview_bundle.scope=%s\nreview_bundle.state=%s\nreview_bundle.subject.patch_digest=%s\nreview_bundle.subject.tree_digest=%s\nreview_bundle.criteria=%d\nreview_bundle.tools=%d\nreview_bundle.evidence=%d\n", bundle.BundleID, bundle.BundleDigest, bundle.Payload.Scope.Ref, bundle.State, bundle.Payload.Subject.PatchDigest, bundle.Payload.Subject.TreeDigest, len(bundle.Payload.Plan.Criteria), len(bundle.Payload.Plan.Tools), len(bundle.Payload.Evidence))
		if explain {
			for _, input := range bundle.Payload.Scope.Sections {
				fmt.Fprintf(stdout, "include.%s=%s reason:%s\n", input.Kind, input.Path, input.Reason)
			}
			for _, entry := range bundle.Payload.Subject.Entries {
				path := entry.Path
				if entry.NewPath != "" {
					path = entry.NewPath
				}
				fmt.Fprintf(stdout, "include.subject=%s class:%s digest:%s reason:%s\n", path, entry.Class, entry.Digest, entry.Reason)
			}
			for _, input := range bundle.Payload.ConsumedInputs {
				fmt.Fprintf(stdout, "include.%s=%s reason:%s\n", input.Kind, input.Path, input.Reason)
			}
			for _, input := range bundle.ExcludedInputs {
				fmt.Fprintf(stdout, "exclude.%s=%s reason:%s\n", input.Kind, input.Path, input.Reason)
			}
		}
		for _, warning := range bundle.Warnings {
			fmt.Fprintf(stdout, "[WARN] %s\n", warning)
		}
		for _, blocker := range bundle.Blockers {
			fmt.Fprintf(stderr, "[ERROR] %s\n", blocker)
		}
	}
	if err != nil || len(bundle.Blockers) > 0 {
		return 1
	}
	for _, input := range bundle.ExcludedInputs {
		if input.Kind == "derived" || input.Kind == "derived-section" || input.Kind == "lifecycle" {
			noteUsageSignals(stdout, "false-staleness-avoided")
			break
		}
	}
	return 0
}

func cmdReviewVerify(root string, args []string, stdout, stderr io.Writer) int {
	jsonOutput, target := false, ""
	for _, arg := range args {
		if arg == "--json" {
			jsonOutput = true
		} else if strings.HasPrefix(arg, "-") || target != "" {
			fmt.Fprintln(stderr, "Usage: pose review verify <scope|bundle-id|bundle-path> [--json]")
			return 2
		} else {
			target = arg
		}
	}
	if target == "" {
		fmt.Fprintln(stderr, "Usage: pose review verify <scope|bundle-id|bundle-path> [--json]")
		return 2
	}
	store := posemodel.Store{Root: root}
	ref, err := resolveReviewBundleScope(store, target)
	if err != nil {
		fmt.Fprintf(stderr, "pose review verify: %v\n", err)
		return 1
	}
	verification, err := store.VerifyReviewBundle(ref)
	if err != nil {
		fmt.Fprintf(stderr, "pose review verify: %v\n", err)
		return 1
	}
	if jsonOutput {
		if code := writeJSON(stdout, verification); code != 0 {
			return code
		}
	} else {
		fmt.Fprintf(stdout, "review_verify.scope=%s\nreview_verify.state=%s\nreview_verify.fresh=%t\nreview_verify.approved=%t\nreview_verify.next_action=%s\n", verification.Scope, verification.State, verification.Fresh, verification.Approved, verification.NextAction)
		for _, warning := range verification.Warnings {
			fmt.Fprintf(stdout, "[WARN] %s\n", warning)
		}
		for _, blocker := range verification.Blockers {
			fmt.Fprintf(stderr, "[ERROR] %s\n", blocker)
		}
	}
	if verification.State == "superseded" {
		noteUsageSignals(stdout, "supersession")
	}
	if verification.Attestation != nil && len(verification.Attestation.ReusedFrom) > 0 {
		noteUsageSignals(stdout, "criterion-reuse")
	}
	if !verification.Approved {
		return 1
	}
	return 0
}

func resolveReviewBundleScope(store posemodel.Store, target string) (string, error) {
	if _, err := posemodel.ParseScopeRef(target); err == nil {
		return target, nil
	}
	id := target
	if strings.HasSuffix(target, ".json") {
		clean := filepath.Clean(filepath.FromSlash(target))
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
			return "", fmt.Errorf("bundle path escapes project root")
		}
		id = strings.TrimSuffix(filepath.Base(clean), ".json")
	}
	bundle, err := store.LoadReviewBundle(id)
	if err != nil {
		return "", err
	}
	return bundle.Payload.Scope.Ref, nil
}

func cmdReviewAttest(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) >= 2 && args[0] == "--envelope" {
		apply := false
		if len(args) == 3 && args[2] == "--apply" {
			apply = true
		} else if len(args) != 2 {
			fmt.Fprintln(stderr, "Usage: pose review attest --envelope <project-relative-path> [--apply]")
			return 2
		}
		attestation, err := (posemodel.Store{Root: root}).ImportReviewAttestationEnvelope(args[1], apply)
		if err != nil {
			fmt.Fprintf(stderr, "pose review attest: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "review_attestation.envelope=verified\nreview_attestation.id=%s\nreview_attestation.bundle_id=%s\nreview_attestation.apply=%t\n", attestation.AttestationID, attestation.BundleID, apply)
		return 0
	}
	var target, reviewer, decision, expectedPlanDigest string
	var evidence, findings, rawTools []string
	apply := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--reviewer", "--decision", "--evidence", "--finding", "--tool", "--plan-digest":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "pose review attest: missing option value")
				return 2
			}
			i++
			switch args[i-1] {
			case "--reviewer":
				reviewer = args[i]
			case "--decision":
				decision = args[i]
			case "--evidence":
				evidence = append(evidence, args[i])
			case "--finding":
				findings = append(findings, args[i])
			case "--tool":
				rawTools = append(rawTools, args[i])
			case "--plan-digest":
				expectedPlanDigest = args[i]
			}
		case "--apply":
			apply = true
		default:
			if strings.HasPrefix(args[i], "-") || target != "" {
				fmt.Fprintf(stderr, "pose review attest: unexpected argument %q\n", args[i])
				return 2
			}
			target = args[i]
		}
	}
	if target == "" || reviewer == "" || decision == "" || len(evidence) == 0 {
		fmt.Fprintln(stderr, "pose review attest: bundle, reviewer, decision and at least one evidence ref are required")
		return 2
	}
	store := posemodel.Store{Root: root}
	ref, err := resolveReviewBundleScope(store, target)
	if err != nil {
		fmt.Fprintf(stderr, "pose review attest: %v\n", err)
		return 1
	}
	bundle, err := store.CurrentReviewBundle(ref)
	if err != nil || bundle == nil {
		if err == nil {
			err = fmt.Errorf("scope has no current sealed bundle")
		}
		fmt.Fprintf(stderr, "pose review attest: %v\n", err)
		return 1
	}
	if strings.HasPrefix(target, "rvb-") && target != bundle.BundleID {
		fmt.Fprintln(stderr, "pose review attest: bundle is superseded by current semantic inputs")
		return 1
	}
	if expectedPlanDigest != "" && expectedPlanDigest != bundle.Payload.Plan.PlanDigest {
		fmt.Fprintf(stderr, "pose review attest: expected plan digest %s, current is %s\n", expectedPlanDigest, bundle.Payload.Plan.PlanDigest)
		return 1
	}
	plan := posemodel.ReviewPlan{Criteria: bundle.Payload.Plan.Criteria, Tools: bundle.Payload.Plan.Tools}
	tools, err := reviewToolDispositions(plan, rawTools, true)
	if err != nil {
		fmt.Fprintf(stderr, "pose review attest: %v\n", err)
		return 2
	}
	parsedFindings, err := parseCLIReviewFindings(findings)
	if err != nil {
		fmt.Fprintf(stderr, "pose review attest: %v\n", err)
		return 2
	}
	sort.Strings(evidence)
	criteria := make([]posemodel.ReviewCriterion, 0, len(plan.Criteria))
	for _, criterion := range plan.Criteria {
		if criterion.Required {
			criteria = append(criteria, posemodel.ReviewCriterion{ID: criterion.ID, Disposition: "passed", Evidence: reviewCriterionEvidence(criterion, evidence)})
		}
	}
	att := posemodel.ReviewAttestation{BundleID: bundle.BundleID, BundleDigest: bundle.BundleDigest, Reviewer: reviewer, Decision: decision, Criteria: criteria, Tools: tools, EvidenceRefs: evidence, Findings: parsedFindings}
	if !apply {
		fmt.Fprintf(stdout, "review_attestation.plan=record\nreview_attestation.bundle_id=%s\nreview_attestation.bundle_digest=%s\nreview_attestation.plan_digest=%s\nreview_attestation.apply=false\n", bundle.BundleID, bundle.BundleDigest, bundle.Payload.Plan.PlanDigest)
		return 0
	}
	att, err = store.RecordReviewAttestation(att, time.Now())
	if err != nil {
		fmt.Fprintf(stderr, "pose review attest: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Review attestation recorded: %s\n", filepath.Join(root, filepath.FromSlash(att.Path)))
	return 0
}

func cmdReviewAutoAttest(root string, args []string, stdout, stderr io.Writer) int {
	var target, reviewer string
	apply := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--reviewer":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "pose review auto-attest: missing option value for --reviewer")
				return 2
			}
			i++
			reviewer = args[i]
		case "--apply":
			apply = true
		default:
			if strings.HasPrefix(args[i], "-") || target != "" {
				fmt.Fprintf(stderr, "pose review auto-attest: unexpected argument %q\n", args[i])
				return 2
			}
			target = args[i]
		}
	}
	if target == "" {
		fmt.Fprintln(stderr, "Usage: pose review auto-attest <bundle-id|scope-ref> [--reviewer <id>] [--apply]")
		return 2
	}
	if reviewer == "" {
		reviewer = "agent:auto-attest"
	}
	store := posemodel.Store{Root: root}
	ref, err := resolveReviewBundleScope(store, target)
	if err != nil {
		fmt.Fprintf(stderr, "pose review auto-attest: %v\n", err)
		return 1
	}
	bundle, err := store.CurrentReviewBundle(ref)
	if err != nil || bundle == nil {
		if err == nil {
			err = fmt.Errorf("scope %s has no current sealed bundle", ref)
		}
		fmt.Fprintf(stderr, "pose review auto-attest: %v\n", err)
		return 1
	}
	if strings.HasPrefix(target, "rvb-") && target != bundle.BundleID {
		fmt.Fprintf(stderr, "pose review auto-attest: bundle %s is superseded by %s\n", target, bundle.BundleID)
		return 1
	}
	att, err := store.AutoAttestReviewBundle(bundle.BundleID, reviewer, apply, time.Now())
	if err != nil {
		fmt.Fprintf(stderr, "pose review auto-attest: %v\n", err)
		return 1
	}
	if !apply {
		fmt.Fprintf(stdout, "review_attestation.plan=auto-attest\nreview_attestation.bundle_id=%s\nreview_attestation.bundle_digest=%s\nreview_attestation.reviewer=%s\nreview_attestation.criteria_count=%d\nreview_attestation.tools_count=%d\nreview_attestation.apply=false\n", bundle.BundleID, bundle.BundleDigest, reviewer, len(att.Criteria), len(att.Tools))
		return 0
	}
	fmt.Fprintf(stdout, "Review attestation auto-recorded: %s\n", filepath.Join(root, filepath.FromSlash(att.Path)))
	return 0
}

func parseCLIReviewFindings(values []string) ([]posemodel.ReviewFinding, error) {
	result := make([]posemodel.ReviewFinding, 0, len(values))
	for _, raw := range values {
		parts := strings.Split(raw, "|")
		if len(parts) < 5 || posemodel.ValidateSlug(parts[0]) != nil {
			return nil, fmt.Errorf("finding must be ID|severity|disposition|action|evidence[|owner|rationale|review-by]")
		}
		finding := posemodel.ReviewFinding{ID: parts[0], Severity: parts[1], Disposition: parts[2], Action: parts[3], Evidence: parts[4]}
		if len(parts) > 5 {
			finding.Owner = parts[5]
		}
		if len(parts) > 6 {
			finding.Rationale = parts[6]
		}
		if len(parts) > 7 {
			finding.ReviewBy = parts[7]
		}
		result = append(result, finding)
	}
	return result, nil
}

func cmdReviewRecord(root string, args []string, stdout, stderr io.Writer) int {
	var ref, reviewer, decision, expectedPlanDigest string
	var evidence, findings, toolDispositions []string
	apply := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--reviewer", "--decision", "--evidence", "--finding", "--plan-digest", "--tool":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "pose review record: missing option value")
				return 2
			}
			i++
			switch args[i-1] {
			case "--reviewer":
				reviewer = args[i]
			case "--decision":
				decision = args[i]
			case "--evidence":
				evidence = append(evidence, args[i])
			case "--finding":
				findings = append(findings, args[i])
			case "--plan-digest":
				expectedPlanDigest = args[i]
			case "--tool":
				toolDispositions = append(toolDispositions, args[i])
			}
		case "--apply":
			apply = true
		default:
			if strings.HasPrefix(args[i], "-") || ref != "" {
				fmt.Fprintf(stderr, "pose review record: unexpected argument %q\n", args[i])
				return 2
			}
			ref = args[i]
		}
	}
	if ref == "" || reviewer == "" || decision == "" || len(evidence) == 0 {
		fmt.Fprintln(stderr, "pose review record: scope, reviewer, decision and at least one evidence ref are required")
		return 2
	}
	allowedDecision := map[string]bool{"approved": true, "approved-with-reservations": true, "changes-requested": true, "rejected": true}
	if !allowedDecision[decision] || strings.ContainsAny(reviewer, "\r\n") {
		fmt.Fprintln(stderr, "pose review record: invalid decision or reviewer")
		return 2
	}
	store := posemodel.Store{Root: root}
	policy, err := store.GetReviewPolicy()
	if err != nil {
		fmt.Fprintf(stderr, "pose review record: %v\n", err)
		return 1
	}
	if policy.ReviewBundles {
		verification, verifyErr := store.VerifyReviewBundle(ref)
		if verifyErr != nil {
			fmt.Fprintf(stderr, "pose review record: %v\n", verifyErr)
			return 1
		}
		if verification.Bundle == nil || !verification.Fresh {
			fmt.Fprintln(stderr, "pose review record: bundle policy requires a current sealed bundle; run pose review bundle <scope> --seal")
			return 1
		}
		adapted := append([]string{}, args...)
		for i, value := range adapted {
			if value == ref {
				adapted[i] = verification.Bundle.BundleID
				break
			}
		}
		return cmdReviewAttest(root, adapted, stdout, stderr)
	}
	profile, err := store.ReviewProfileForScope(ref)
	if err != nil {
		fmt.Fprintf(stderr, "pose review record: %v\n", err)
		return 1
	}
	digest, err := store.ScopeDigest(ref)
	if err != nil {
		fmt.Fprintf(stderr, "pose review record: %v\n", err)
		return 1
	}
	plan, err := store.ReviewPlan(ref)
	if err != nil {
		fmt.Fprintf(stderr, "pose review record: %v\n", err)
		return 1
	}
	if len(plan.Blockers) > 0 {
		fmt.Fprintf(stderr, "pose review record: effective review plan is blocked: %s\n", strings.Join(plan.Blockers, "; "))
		return 1
	}
	if expectedPlanDigest != "" && expectedPlanDigest != plan.PlanDigest {
		fmt.Fprintf(stderr, "pose review record: expected plan digest %s, current is %s\n", expectedPlanDigest, plan.PlanDigest)
		return 1
	}
	attempts, err := store.ListReviewAttempts(ref)
	if err != nil {
		fmt.Fprintf(stderr, "pose review record: %v\n", err)
		return 1
	}
	supersedes := ""
	if len(attempts) > 0 {
		supersedes = attempts[len(attempts)-1].ReviewID
	}
	now := time.Now().UTC().Truncate(time.Second)
	sum := sha256.Sum256([]byte(ref + digest + reviewer + now.Format(time.RFC3339)))
	reviewID := "rvw-" + now.Format("20060102T150405Z") + "-" + hex.EncodeToString(sum[:4])
	sort.Strings(evidence)
	content, err := renderReviewAttempt(reviewID, ref, digest, plan, profile, reviewer, decision, now.Format(time.RFC3339), supersedes, evidence, toolDispositions, findings, policy.SchemaVersion >= 2 && policy.ComponentAware)
	if err != nil {
		fmt.Fprintf(stderr, "pose review record: %v\n", err)
		return 2
	}
	path := filepath.Join(root, ".pose", "reviews", reviewID+".md")
	if !apply {
		fmt.Fprintf(stdout, "review.plan=record\nreview.id=%s\nreview.scope=%s\nreview.digest=%s\nreview.plan_digest=%s\nreview.path=%s\nreview.apply=false\n", reviewID, ref, digest, plan.PlanDigest, filepath.ToSlash(strings.TrimPrefix(path, root+string(os.PathSeparator))))
		return 0
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintf(stderr, "pose review record: %v\n", err)
		return 1
	}
	if err := writeFileExclusive(path, []byte(content)); err != nil {
		fmt.Fprintf(stderr, "pose review record: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Review recorded: %s\n", path)
	return 0
}

func renderReviewAttempt(id, scope, digest string, plan posemodel.ReviewPlan, profile posemodel.ReviewProfile, reviewer, decision, reviewedAt, supersedes string, evidence, rawTools, findings []string, componentAware bool) (string, error) {
	tools, err := reviewToolDispositions(plan, rawTools, componentAware)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "---\nschema_version: 1\nreview_id: %s\nscope: %s\nscope_digest: %s\nplan_digest: %s\nprofile: %s\nreviewer: %s\ndecision: %s\nreviewed_at: %s\n", id, scope, digest, plan.PlanDigest, profile.Ref(), reviewer, decision, reviewedAt)
	if supersedes == "" {
		b.WriteString("supersedes:\n")
	} else {
		fmt.Fprintf(&b, "supersedes: %s\n", supersedes)
	}
	fmt.Fprintf(&b, "evidence_refs: [%s]\n---\n\n## Criteria\n", strings.Join(evidence, ", "))
	for _, criterion := range plan.Criteria {
		if !criterion.Required {
			continue
		}
		fmt.Fprintf(&b, "- %s [passed] evidence:%s\n", criterion.ID, reviewCriterionEvidence(criterion, evidence))
	}
	b.WriteString("\n## Tools\n")
	for _, tool := range tools {
		fmt.Fprintf(&b, "- %s [%s]", tool.ID, tool.Disposition)
		if tool.Component != "" {
			fmt.Fprintf(&b, " component:%s", url.QueryEscape(tool.Component))
		}
		if tool.Evidence != "" {
			fmt.Fprintf(&b, " evidence:%s", tool.Evidence)
		}
		if tool.Rationale != "" {
			fmt.Fprintf(&b, " rationale:%s", strings.ReplaceAll(tool.Rationale, " ", "_"))
		}
		b.WriteByte('\n')
	}
	b.WriteString("\n## Findings\n")
	for _, raw := range findings {
		parts := strings.Split(raw, "|")
		if len(parts) < 5 || posemodel.ValidateSlug(parts[0]) != nil {
			return "", fmt.Errorf("finding must be ID|severity|disposition|action|evidence")
		}
		fmt.Fprintf(&b, "- %s [%s] severity:%s action:%s evidence:%s", parts[0], parts[2], parts[1], strings.ReplaceAll(parts[3], " ", "_"), strings.ReplaceAll(parts[4], " ", "_"))
		if len(parts) > 5 && parts[5] != "" {
			fmt.Fprintf(&b, " owner:%s", parts[5])
		}
		if len(parts) > 6 && parts[6] != "" {
			fmt.Fprintf(&b, " rationale:%s", strings.ReplaceAll(parts[6], " ", "_"))
		}
		if len(parts) > 7 && parts[7] != "" {
			fmt.Fprintf(&b, " review_by:%s", parts[7])
		}
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func reviewToolDispositions(plan posemodel.ReviewPlan, raw []string, componentAware bool) ([]posemodel.ReviewToolDisposition, error) {
	if !componentAware {
		if len(raw) > 0 {
			return nil, fmt.Errorf("tool dispositions require component-aware review policy")
		}
		return nil, nil
	}
	planned := map[string]posemodel.ReviewPlanTool{}
	for _, tool := range plan.Tools {
		planned[cliReviewToolKey(tool.ID, tool.Component)] = tool
	}
	provided := map[string]posemodel.ReviewToolDisposition{}
	for _, value := range raw {
		parts := strings.Split(value, "|")
		if len(parts) < 4 || len(parts) > 5 || posemodel.ValidateSlug(parts[0]) != nil {
			return nil, fmt.Errorf("tool must be ID|component|disposition|evidence|rationale")
		}
		component := parts[1]
		if component == "-" {
			component = ""
		} else if decoded, err := url.QueryUnescape(component); err == nil {
			component = decoded
		}
		disposition := posemodel.ReviewToolDisposition{ID: parts[0], Component: component, Disposition: parts[2], Evidence: parts[3]}
		if len(parts) == 5 {
			disposition.Rationale = parts[4]
		}
		key := cliReviewToolKey(disposition.ID, disposition.Component)
		tool, ok := planned[key]
		if !ok {
			return nil, fmt.Errorf("tool disposition does not match effective plan: %s", cliReviewToolLabel(disposition.ID, disposition.Component))
		}
		if _, duplicate := provided[key]; duplicate {
			return nil, fmt.Errorf("duplicate tool disposition: %s", cliReviewToolLabel(disposition.ID, disposition.Component))
		}
		if err := validateCLIReviewToolDisposition(tool, disposition); err != nil {
			return nil, err
		}
		provided[key] = disposition
	}

	result := make([]posemodel.ReviewToolDisposition, 0, len(plan.Tools))
	for _, tool := range plan.Tools {
		key := cliReviewToolKey(tool.ID, tool.Component)
		if disposition, ok := provided[key]; ok {
			result = append(result, disposition)
			continue
		}
		if cliReviewToolHasPrecondition(tool, "review-complete") {
			result = append(result, posemodel.ReviewToolDisposition{ID: tool.ID, Component: tool.Component, Disposition: "deferred", Rationale: "post-review gate"})
			continue
		}
		if tool.Requiredness == "recommended" {
			result = append(result, posemodel.ReviewToolDisposition{ID: tool.ID, Component: tool.Component, Disposition: "not-used", Rationale: "not used during review"})
			continue
		}
		return nil, fmt.Errorf("required review tool %s needs --tool evidence", cliReviewToolLabel(tool.ID, tool.Component))
	}
	return result, nil
}

func validateCLIReviewToolDisposition(tool posemodel.ReviewPlanTool, disposition posemodel.ReviewToolDisposition) error {
	label := cliReviewToolLabel(tool.ID, tool.Component)
	completion := cliReviewToolHasPrecondition(tool, "review-complete")
	switch disposition.Disposition {
	case "passed", "failed":
		parts := strings.SplitN(disposition.Evidence, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("%s review tool %s requires an evidence ref", disposition.Disposition, label)
		}
		if len(tool.EvidenceClasses) > 0 && !containsCLIReviewValue(tool.EvidenceClasses, parts[0]) {
			return fmt.Errorf("review tool %s requires evidence class %s", label, strings.Join(tool.EvidenceClasses, ","))
		}
	case "not-used":
		if tool.Requiredness == "required" {
			return fmt.Errorf("required review tool %s cannot be not-used", label)
		}
		if disposition.Rationale == "" {
			return fmt.Errorf("recommended review tool %s needs a not-used rationale", label)
		}
	case "deferred":
		if !completion || disposition.Rationale == "" {
			return fmt.Errorf("review tool %s cannot be deferred without a post-review rationale", label)
		}
	default:
		return fmt.Errorf("review tool %s has invalid disposition %q", label, disposition.Disposition)
	}
	return nil
}

func cliReviewToolHasPrecondition(tool posemodel.ReviewPlanTool, expected string) bool {
	return containsCLIReviewValue(tool.Preconditions, expected)
}

func containsCLIReviewValue(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func cliReviewToolKey(id, component string) string {
	return id + "\x00" + component
}

func cliReviewToolLabel(id, component string) string {
	if component == "" {
		return id
	}
	return id + " (component " + component + ")"
}

func reviewCriterionEvidence(criterion posemodel.ReviewPlanCriterion, evidence []string) string {
	for _, class := range criterion.EvidenceClasses {
		prefix := class + ":"
		for _, ref := range evidence {
			if strings.HasPrefix(ref, prefix) {
				return ref
			}
		}
	}
	return evidence[0]
}

func writeFileExclusive(path string, content []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(content); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func cmdClose(root string, args []string, stdout, stderr io.Writer) int {
	ref, _, ok := parseScopeCheckArgs("close", args, stderr)
	if !ok {
		return 2
	}
	store := posemodel.Store{Root: root}
	state, err := store.GetCloseoutState(ref)
	if err != nil {
		fmt.Fprintf(stderr, "pose close: %v\n", err)
		return 1
	}
	if !state.Review.Approved || len(state.Children) > 0 && hasOpenChild(state.Children) {
		fmt.Fprintf(stderr, "pose close: scope is not eligible; next action: %s\n", state.NextAction)
		return 1
	}
	scope, _ := posemodel.ParseScopeRef(ref)
	if scope.Kind == "milestone" {
		fmt.Fprintf(stdout, "Milestone closeout verified: %s\n", ref)
		return 0
	}
	var path string
	if scope.Kind == "spec" {
		sp, err := store.GetSpec(scope.Slug)
		if err != nil {
			fmt.Fprintf(stderr, "pose close: %v\n", err)
			return 1
		}
		if policy, err := posemodel.LoadArtifactPolicy(root); err != nil {
			fmt.Fprintf(stderr, "pose close: %v\n", err)
			return 1
		} else if policy.Enabled && sp.CreatedAt >= policy.AdoptedAt {
			claims, found, err := posemodel.ParseArtifactClaims(*sp, policy)
			if err != nil || !found || len(claims) == 0 {
				fmt.Fprintf(stderr, "pose close: artifact declaration gate failed: %v\n", err)
				PrintContributorFailureHint(root, stderr, cliLocaleValue())
				return 1
			}
		}
		if policy, err := posemodel.LoadDeliveryPolicy(root); err != nil {
			fmt.Fprintf(stderr, "pose close: %v\n", err)
			return 1
		} else if policy.Enabled && sp.CreatedAt >= policy.AdoptedAt {
			if blockers := deliverySpecBlockers(root, sp.Slug); len(blockers) > 0 {
				fmt.Fprintf(stderr, "pose close: delivery assurance gate failed: %s\n", strings.Join(blockers, "; "))
				PrintContributorFailureHint(root, stderr, cliLocaleValue())
				return 1
			}
		}
		path = sp.Path
	} else {
		if policy, err := posemodel.LoadDeliveryPolicy(root); err != nil {
			fmt.Fprintf(stderr, "pose close: %v\n", err)
			return 1
		} else if policy.Enabled {
			var gateOut, gateErr bytes.Buffer
			if code := cmdRoadmapCheck(root, []string{scope.Slug, "--strict"}, &gateOut, &gateErr); code != 0 {
				fmt.Fprintf(stderr, "pose close: roadmap delivery gate failed: %s%s", gateErr.String(), gateOut.String())
				return 1
			}
		}
		path = filepath.Join(root, ".pose", "roadmaps", scope.Slug+".md")
	}
	if err := applyLifecycleDone(path, scope.Kind == "spec"); err != nil {
		fmt.Fprintf(stderr, "pose close: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Closed: %s\n", ref)
	return 0
}

type continuousCloseoutSelection struct {
	SchemaVersion int    `json:"schema_version"`
	Scope         string `json:"scope"`
	StartedAt     string `json:"started_at"`
	Status        string `json:"status"`
}

func cmdContinuousCloseout(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		args = []string{"status"}
	}
	path := filepath.Join(root, ".pose", "continuous-closeout.json")
	switch args[0] {
	case "start":
		if len(args) < 2 || len(args) > 3 || len(args) == 3 && args[2] != "--apply" {
			fmt.Fprintln(stderr, "Usage: pose continuous-closeout start <scope> [--apply]")
			return 2
		}
		ref := args[1]
		store := posemodel.Store{Root: root}
		policy, err := store.GetReviewPolicy()
		if err != nil || !policy.Enabled || !policy.ContinuousCloseout {
			fmt.Fprintln(stderr, "pose continuous-closeout: continuous mode is not enabled by review policy")
			return 1
		}
		if _, err := store.GetCloseoutState(ref); err != nil {
			fmt.Fprintf(stderr, "pose continuous-closeout: %v\n", err)
			return 1
		}
		selection := continuousCloseoutSelection{SchemaVersion: 1, Scope: ref, StartedAt: time.Now().UTC().Truncate(time.Second).Format(time.RFC3339), Status: "active"}
		if len(args) == 2 {
			fmt.Fprintf(stdout, "continuous.scope=%s\ncontinuous.apply=false\n", ref)
			return 0
		}
		raw, _ := json.MarshalIndent(selection, "", "  ")
		raw = append(raw, '\n')
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			fmt.Fprintf(stderr, "pose continuous-closeout: %v\n", err)
			return 1
		}
		if _, err := os.Stat(path); err == nil {
			fmt.Fprintln(stderr, "pose continuous-closeout: an active terminal scope already exists")
			return 1
		}
		if err := writeFileExclusive(path, raw); err != nil {
			fmt.Fprintf(stderr, "pose continuous-closeout: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Continuous closeout started: %s\n", ref)
		return 0
	case "status":
		raw, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			fmt.Fprintln(stdout, "continuous.active=false")
			return 0
		}
		if err != nil {
			fmt.Fprintf(stderr, "pose continuous-closeout: %v\n", err)
			return 1
		}
		var selection continuousCloseoutSelection
		if err := json.Unmarshal(raw, &selection); err != nil || selection.SchemaVersion != 1 || selection.Status != "active" {
			fmt.Fprintln(stderr, "pose continuous-closeout: invalid persisted selection")
			return 1
		}
		state, err := (posemodel.Store{Root: root}).GetCloseoutState(selection.Scope)
		if err != nil {
			fmt.Fprintf(stderr, "pose continuous-closeout: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "continuous.active=true\ncontinuous.scope=%s\ncontinuous.terminal=%t\ncontinuous.next_action=%s\n", selection.Scope, state.Terminal, state.NextAction)
		if !state.Terminal {
			return 1
		}
		return 0
	case "complete":
		if len(args) != 2 || args[1] != "--apply" {
			fmt.Fprintln(stderr, "Usage: pose continuous-closeout complete --apply")
			return 2
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(stderr, "pose continuous-closeout: %v\n", err)
			return 1
		}
		var selection continuousCloseoutSelection
		if json.Unmarshal(raw, &selection) != nil {
			fmt.Fprintln(stderr, "pose continuous-closeout: invalid persisted selection")
			return 1
		}
		state, err := (posemodel.Store{Root: root}).GetCloseoutState(selection.Scope)
		if err != nil || !state.Terminal {
			fmt.Fprintf(stderr, "pose continuous-closeout: terminal success is not satisfied; next action: %s\n", state.NextAction)
			return 1
		}
		if err := os.Remove(path); err != nil {
			fmt.Fprintf(stderr, "pose continuous-closeout: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Continuous closeout completed: %s\n", selection.Scope)
		return 0
	default:
		fmt.Fprintln(stderr, "Usage: pose continuous-closeout <start|status|complete> [...]")
		return 2
	}
}

func hasOpenChild(children []posemodel.CloseoutState) bool {
	for _, child := range children {
		if !child.Terminal {
			return true
		}
	}
	return false
}

func applyLifecycleDone(path string, spec bool) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(raw)
	if !strings.HasPrefix(text, "---\n") {
		return fmt.Errorf("artifact has no flat frontmatter")
	}
	text = replaceFrontmatterValue(text, "status", "done")
	if spec {
		text = replaceFrontmatterValue(text, "completed_at", time.Now().UTC().Format("2006-01-02"))
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".pose-close-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.WriteString(text); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func replaceFrontmatterValue(content, key, value string) string {
	end := strings.Index(content[4:], "\n---")
	if end < 0 {
		return content
	}
	end += 4
	head, tail := content[:end], content[end:]
	lines := strings.Split(head, "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, key+":") {
			lines[i] = key + ": " + value
			found = true
		}
	}
	if !found {
		lines = append(lines, key+": "+value)
	}
	return strings.Join(lines, "\n") + tail
}
