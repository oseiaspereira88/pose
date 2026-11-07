package pose

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type governanceReadViewIntegrationCapture struct {
	call int32
	data string
}

type governanceReadViewIntegrationResult struct {
	report ObligationReport
	err    error
}

func TestGovernanceReadViewProjectionRetriesOnceAndReturnsOnlyStableAttempt(t *testing.T) {
	governanceReadViewProbeUseKnownEngineVersion(t)
	store := obligationFixture(t)
	path := filepath.Join(store.Root, ".pose", "operation-generation.txt")
	governanceReadViewIntegrationWriteGeneration(t, path, "a", time.Time{})
	initialInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	previous := actionRequestObligationSource
	var calls atomic.Int32
	captured := make(chan governanceReadViewIntegrationCapture, 3)
	releaseFirst := make(chan struct{})
	var releaseOnce atomic.Bool
	t.Cleanup(func() {
		if releaseOnce.CompareAndSwap(false, true) {
			close(releaseFirst)
		}
		actionRequestObligationSource = previous
	})
	actionRequestObligationSource = func(s Store, project string, _ []Spec, _ bool, _ *Coverage) ([]Obligation, error) {
		call := calls.Add(1)
		raw, err := s.fsys().ReadFile(path)
		if err != nil {
			return nil, err
		}
		value := string(raw)
		captured <- governanceReadViewIntegrationCapture{call: call, data: value}
		if call == 1 {
			<-releaseFirst
		}
		return []Obligation{governanceReadViewIntegrationObligation(project, value)}, nil
	}

	result := governanceReadViewIntegrationProjectAsync(store)
	first := governanceReadViewIntegrationAwaitCapture(t, captured)
	if first.call != 1 || first.data != "a" {
		t.Fatalf("first producer capture = %+v, want call 1 with generation a", first)
	}
	governanceReadViewIntegrationWriteGeneration(t, path, "b", initialInfo.ModTime())
	if releaseOnce.CompareAndSwap(false, true) {
		close(releaseFirst)
	}

	completed := governanceReadViewIntegrationAwaitResult(t, result)
	if completed.err != nil {
		t.Fatal(completed.err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("producer ran %d times; a changed first attempt must cause exactly one retry", got)
	}
	second := governanceReadViewIntegrationAwaitCapture(t, captured)
	if second.call != 2 || second.data != "b" {
		t.Fatalf("retry capture = %+v, want call 2 with generation b", second)
	}
	actionObligations := governanceReadViewIntegrationObligationsFrom(completed.report, "action-requests")
	if len(actionObligations) != 1 || actionObligations[0].Message != "captured generation b" {
		t.Fatalf("the report did not discard the first attempt and return the stable retry: %+v", actionObligations)
	}
	if completed.report.Snapshot.ReadView == nil {
		t.Fatal("stable retry omitted read-view metadata")
	}
	for _, limitation := range completed.report.Snapshot.ReadView.Limitations {
		if strings.Contains(limitation, "producer action-requests:") {
			t.Fatalf("stable retry left the action-request closure limited: %+v", completed.report.Snapshot.ReadView)
		}
	}
	if !completed.report.Snapshot.Coherent {
		t.Fatalf("stable retry returned an incoherent snapshot: %+v", completed.report.Snapshot)
	}
	if state := governanceReadViewIntegrationCoverage(completed.report, "action-requests"); state != CoverageStateCurrent {
		t.Fatalf("stable retry action-request coverage = %q, want current", state)
	}
	independent, err := store.ProjectObligations(ObligationQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("independent stable projection brought producer call count to %d, want 3 total", got)
	}
	third := governanceReadViewIntegrationAwaitCapture(t, captured)
	if third.call != 3 || third.data != "b" {
		t.Fatalf("independent stable capture = %+v, want call 3 with generation b", third)
	}
	if independent.Snapshot.ReadView == nil || independent.Snapshot.ReadView.ID != completed.report.Snapshot.ReadView.ID {
		var independentID string
		if independent.Snapshot.ReadView != nil {
			independentID = independent.Snapshot.ReadView.ID
		}
		t.Fatalf("retry identity does not match an independent stable projection of generation b: retry=%q independent=%q", completed.report.Snapshot.ReadView.ID, independentID)
	}
}

func TestGovernanceReadViewUsesUniqueAuthorizedProjectForPhysicalRoot(t *testing.T) {
	root := obligationFixture(t).Root
	const projectID = "coordinator"
	resolver := ArtifactResolver{Roots: NewRootsIn(NativeFiles(), RootsConfig{DefaultRoot: root, DefaultProjectID: projectID})}
	store := Store{Root: root, FederatedResolver: &resolver}
	base, view := store.beginGovernanceReadView()
	if base.FederatedProjectID != projectID || view.operation.mainProject != projectID || view.operation.roots[0].project != projectID {
		t.Fatalf("read view guessed a directory-derived identity instead of the unique resolver binding: store=%q main=%q root=%q", base.FederatedProjectID, view.operation.mainProject, view.operation.roots[0].project)
	}
	if reasons := view.unknown["projection"]; len(reasons) != 0 {
		t.Fatalf("a unique authorized root identity made the view unknown: %v", reasons)
	}
}

func TestGovernanceReadViewRefusesAmbiguousAliasesForPhysicalRoot(t *testing.T) {
	root := obligationFixture(t).Root
	resolver := ArtifactResolver{Roots: NewRootsIn(NativeFiles(), RootsConfig{
		DefaultRoot: root, DefaultProjectID: "coordinator",
		Explicit: map[string]string{"proj.alias": root},
	})}
	store := Store{Root: root, FederatedResolver: &resolver}
	base, view := store.beginGovernanceReadView()
	if base.FederatedProjectID != "" || view.operation.mainProject != "" {
		t.Fatalf("read view guessed one of multiple authorized aliases: store=%q main=%q", base.FederatedProjectID, view.operation.mainProject)
	}
	if len(view.unknown["projection"]) == 0 {
		t.Fatalf("ambiguous physical-root identity did not disclose unknown projection coverage: %v", view.unknown)
	}
}

func TestGovernanceReadViewRequiresDerivedIdentityToBeAuthorizedAndBound(t *testing.T) {
	root := obligationFixture(t).Root
	derived := DefaultProjectIDIn(NativeFiles(), root)
	otherRoot := t.TempDir()

	for _, test := range []struct {
		name      string
		authorize func(string) bool
	}{
		{name: "derived identity is not authorized", authorize: func(id string) bool { return id != derived }},
		{name: "derived identity is not bound", authorize: func(string) bool { return true }},
	} {
		t.Run(test.name, func(t *testing.T) {
			resolver := ArtifactResolver{
				Roots:     NewRootsIn(NativeFiles(), RootsConfig{DefaultRoot: otherRoot, DefaultProjectID: "unrelated"}),
				Authorize: test.authorize,
			}
			base, view := (Store{Root: root, FederatedResolver: &resolver}).beginGovernanceReadView()
			if base.FederatedProjectID != "" || view.operation.mainProject != "" {
				t.Fatalf("read view invented a primary identity outside the resolver binding: store=%q main=%q", base.FederatedProjectID, view.operation.mainProject)
			}
			if len(view.unknown["projection"]) == 0 {
				t.Fatalf("unbound derived identity did not leave the read view unknown: %v", view.unknown)
			}
		})
	}
}

func TestGovernanceReadViewRejectsAnExplicitIdentityBoundToAnotherRoot(t *testing.T) {
	root := obligationFixture(t).Root
	otherRoot := t.TempDir()
	const projectID = "coordinator"
	resolver := ArtifactResolver{Roots: NewRootsIn(NativeFiles(), RootsConfig{DefaultRoot: otherRoot, DefaultProjectID: projectID})}
	store := Store{Root: root, FederatedProjectID: projectID, FederatedResolver: &resolver}
	_, view := store.beginGovernanceReadView()
	if len(view.unknown["projection"]) == 0 {
		t.Fatalf("explicit project identity mapped to another physical root was accepted: %v", view.unknown)
	}
}

func TestGovernanceReadViewDoesNotFallbackFromMalformedDeclaredIdentity(t *testing.T) {
	root := obligationFixture(t).Root
	if err := os.WriteFile(filepath.Join(root, ".pose", "project.json"), []byte(`{"schema_version":1,"project_id":"Not A Slug"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	resolver := ArtifactResolver{Roots: NewRootsIn(NativeFiles(), RootsConfig{DefaultRoot: root})}
	base, view := (Store{Root: root, FederatedResolver: &resolver}).beginGovernanceReadView()
	if base.FederatedProjectID != "" || view.operation.mainProject != "" {
		t.Fatalf("malformed declared identity fell back to a directory-derived project: store=%q main=%q", base.FederatedProjectID, view.operation.mainProject)
	}
	if len(view.unknown["projection"]) == 0 {
		t.Fatalf("malformed declared identity did not make the read view unknown: %v", view.unknown)
	}
}

func TestUniqueProjectIdentityRootLookupFiltersAuthorizationBeforeFilesystem(t *testing.T) {
	root := obligationFixture(t).Root
	files := &governanceReadViewRootIdentityFiles{Files: NativeFiles()}
	roots := &Roots{
		byProject: map[string]string{"authorized": root, "denied": root},
		conflicts: map[string]bool{},
		fs:        files,
	}
	project, found, ambiguous, overflow := uniqueProjectIDForRootIn(files, roots, root, func(id string) bool {
		return id == "authorized"
	})
	if !found || ambiguous || overflow || project != "authorized" {
		t.Fatalf("unique authorized binding = %q, found=%v ambiguous=%v overflow=%v", project, found, ambiguous, overflow)
	}
	if files.evalSymlinks != 2 {
		t.Fatalf("root lookup made %d filesystem identity checks; want only the two checks for the authorized candidate", files.evalSymlinks)
	}
}

func TestUniqueProjectIdentityRootLookupFailsClosedBeforeOversizedRegistryScan(t *testing.T) {
	root := t.TempDir()
	byProject := make(map[string]string, governanceReadViewMaxRecords+1)
	for i := 0; i <= governanceReadViewMaxRecords; i++ {
		byProject[fmt.Sprintf("project-%05d", i)] = root
	}
	files := &governanceReadViewRootIdentityFiles{Files: NativeFiles()}
	project, found, ambiguous, overflow := uniqueProjectIDForRootIn(files, &Roots{byProject: byProject, fs: files}, root, func(string) bool { return true })
	if project != "" || found || ambiguous || !overflow {
		t.Fatalf("oversized root registry was not rejected: project=%q found=%v ambiguous=%v overflow=%v", project, found, ambiguous, overflow)
	}
	if files.evalSymlinks != 0 {
		t.Fatalf("oversized root registry triggered %d filesystem identity checks before budget rejection", files.evalSymlinks)
	}
}

type governanceReadViewRootIdentityFiles struct {
	Files
	evalSymlinks int
}

func (f *governanceReadViewRootIdentityFiles) EvalSymlinks(name string) (string, error) {
	f.evalSymlinks++
	return f.Files.EvalSymlinks(name)
}

func TestGovernanceReadViewProjectionStopsAfterOneUnstableRetryAndLocalizesCoverage(t *testing.T) {
	governanceReadViewProbeUseKnownEngineVersion(t)
	store := obligationFixture(t)
	path := filepath.Join(store.Root, ".pose", "operation-generation.txt")
	governanceReadViewIntegrationWriteGeneration(t, path, "a", time.Time{})
	initialInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	previous := actionRequestObligationSource
	var calls atomic.Int32
	captured := make(chan governanceReadViewIntegrationCapture, 3)
	release := []chan struct{}{make(chan struct{}), make(chan struct{})}
	var released [2]atomic.Bool
	t.Cleanup(func() {
		for i := range release {
			if released[i].CompareAndSwap(false, true) {
				close(release[i])
			}
		}
		actionRequestObligationSource = previous
	})
	actionRequestObligationSource = func(s Store, project string, _ []Spec, _ bool, _ *Coverage) ([]Obligation, error) {
		call := calls.Add(1)
		raw, err := s.fsys().ReadFile(path)
		if err != nil {
			return nil, err
		}
		value := string(raw)
		captured <- governanceReadViewIntegrationCapture{call: call, data: value}
		if call <= int32(len(release)) {
			<-release[call-1]
		}
		return []Obligation{governanceReadViewIntegrationObligation(project, value)}, nil
	}

	result := governanceReadViewIntegrationProjectAsync(store)
	first := governanceReadViewIntegrationAwaitCapture(t, captured)
	if first.call != 1 || first.data != "a" {
		t.Fatalf("first producer capture = %+v, want call 1 with generation a", first)
	}
	governanceReadViewIntegrationWriteGeneration(t, path, "b", initialInfo.ModTime())
	if released[0].CompareAndSwap(false, true) {
		close(release[0])
	}

	second := governanceReadViewIntegrationAwaitCapture(t, captured)
	if second.call != 2 || second.data != "b" {
		t.Fatalf("retry capture = %+v, want call 2 with generation b", second)
	}
	governanceReadViewIntegrationWriteGeneration(t, path, "c", initialInfo.ModTime())
	if released[1].CompareAndSwap(false, true) {
		close(release[1])
	}

	completed := governanceReadViewIntegrationAwaitResult(t, result)
	if completed.err != nil {
		t.Fatal(completed.err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("producer ran %d times; projection may retry at most once", got)
	}
	actionObligations := governanceReadViewIntegrationObligationsFrom(completed.report, "action-requests")
	if len(actionObligations) != 1 || actionObligations[0].Message != "captured generation b" {
		t.Fatalf("unstable projection did not preserve only the second attempt's answer: %+v", actionObligations)
	}
	if completed.report.Snapshot.ReadView == nil || completed.report.Snapshot.ReadView.Freshness != ReadViewFreshnessUnknown {
		t.Fatalf("a second mutation was reported as a current read view: %+v", completed.report.Snapshot.ReadView)
	}
	if completed.report.Snapshot.Coherent {
		t.Fatalf("a second mutation was reported as a coherent snapshot: %+v", completed.report.Snapshot)
	}
	if state := governanceReadViewIntegrationCoverage(completed.report, "action-requests"); state != CoverageStateIncomplete {
		t.Fatalf("changed action-request coverage = %q, want incomplete", state)
	}
	if state := governanceReadViewIntegrationCoverage(completed.report, "readiness"); state != CoverageStateCurrent {
		t.Fatalf("unrelated readiness coverage = %q, want current", state)
	}
}

func TestGovernanceReadViewLocalProjectionIgnoresUnavailableUnusedAuthorizedRoot(t *testing.T) {
	governanceReadViewProbeUseKnownEngineVersion(t)
	store := obligationFixture(t)
	const projectID = "proj.local-governance"
	projectIdentity := filepath.Join(store.Root, ".pose", "project.json")
	if err := os.WriteFile(projectIdentity, []byte(`{"schema_version":1,"project_id":"`+projectID+`","name":"local fixture"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	localRoots := NewRootsIn(NativeFiles(), RootsConfig{
		DefaultRoot:      store.Root,
		DefaultProjectID: projectID,
	})
	localResolver := ArtifactResolver{Roots: localRoots, FS: NativeFiles()}
	store.FederatedProjectID = projectID
	store.FederatedResolver = &localResolver
	baseline, err := store.ProjectObligations(ObligationQuery{Scope: "spec:consumer"})
	if err != nil {
		t.Fatal(err)
	}

	unavailableRoot := filepath.Join(t.TempDir(), "not-materialized")
	registeredRoots := NewRootsIn(NativeFiles(), RootsConfig{
		DefaultRoot:      store.Root,
		DefaultProjectID: projectID,
		Explicit:         map[string]string{"proj.unused-unavailable": unavailableRoot},
	})
	registeredResolver := ArtifactResolver{Roots: registeredRoots, FS: NativeFiles()}
	store.FederatedResolver = &registeredResolver

	report, err := store.ProjectObligations(ObligationQuery{Scope: "spec:consumer"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Snapshot.ReadView == nil || baseline.Snapshot.ReadView == nil {
		t.Fatalf("local projection omitted its read-view metadata: report=%+v baseline=%+v", report.Snapshot.ReadView, baseline.Snapshot.ReadView)
	}
	if state := governanceReadViewIntegrationCoverage(report, "readiness"); state != CoverageStateCurrent {
		t.Fatalf("local scoped readiness coverage = %q, want current", state)
	}
	if !reflect.DeepEqual(report.Coverage, baseline.Coverage) {
		t.Fatalf("unused authorized root changed local producer coverage:\nwith root=%+v\nwithout root=%+v", report.Coverage, baseline.Coverage)
	}
	if !reflect.DeepEqual(report.Snapshot.ReadView.Limitations, baseline.Snapshot.ReadView.Limitations) {
		t.Fatalf("unused authorized root changed local read-view limitations:\nwith root=%v\nwithout root=%v", report.Snapshot.ReadView.Limitations, baseline.Snapshot.ReadView.Limitations)
	}
	if !reflect.DeepEqual(report.Snapshot.ReadView.Bindings, baseline.Snapshot.ReadView.Bindings) {
		t.Fatalf("unused authorized root changed the relevant root vector:\nwith root=%+v\nwithout root=%+v", report.Snapshot.ReadView.Bindings, baseline.Snapshot.ReadView.Bindings)
	}
	for _, binding := range report.Snapshot.ReadView.Bindings {
		if binding.ProjectID == "proj.unused-unavailable" {
			t.Fatalf("read view bound an unused unavailable root: %+v", report.Snapshot.ReadView.Bindings)
		}
	}
	foundLocal := false
	for _, binding := range report.Snapshot.ReadView.Bindings {
		foundLocal = foundLocal || binding.ProjectID == projectID
	}
	if !foundLocal {
		t.Fatalf("read view omitted the local root actually used: %+v", report.Snapshot.ReadView.Bindings)
	}
}

func TestGovernanceReadViewOperationSharesInputByteBudgetAcrossRoots(t *testing.T) {
	governanceReadViewProbeUseKnownEngineVersion(t)
	const projectA, projectB = "proj.budget-a", "proj.budget-b"
	rootA := governanceReadViewIntegrationCommittedRoot(t, projectA)
	rootB := governanceReadViewIntegrationCommittedRoot(t, projectB)
	pathA := filepath.Join(rootA, "large-a.bin")
	pathB := filepath.Join(rootB, "large-b.bin")
	// Small real payloads exercise the shared limit without allocating buffers
	// proportional to the production cap. Model prior operation consumption so
	// the two remaining reads exceed the aggregate allowance by two bytes.
	const perRootBytes = int64(33)
	for _, path := range []string{pathA, pathB} {
		if err := os.WriteFile(path, bytes.Repeat([]byte("x"), int(perRootBytes)), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	roots := NewRootsIn(NativeFiles(), RootsConfig{
		DefaultRoot:      rootA,
		DefaultProjectID: projectA,
		Explicit:         map[string]string{projectB: rootB},
	})
	resolver := ArtifactResolver{Roots: roots, FS: NativeFiles()}
	store := Store{Root: rootA, FederatedProjectID: projectA, FederatedResolver: &resolver}
	_, viewA := store.beginGovernanceReadView()
	if viewA.operation == nil {
		t.Fatal("production read-view initializer omitted its operation")
	}
	viewA.operation.budgetMu.Lock()
	viewA.operation.inputBytes = governanceReadViewMaxInputBytes - (2*perRootBytes - 2)
	viewA.operation.budgetMu.Unlock()
	shared := viewA.filesForProject("action-requests", projectA)
	first, err := shared.ReadFile(pathA)
	if err != nil {
		t.Fatalf("first root read (%d bytes) failed: %v", perRootBytes, err)
	}
	if int64(len(first)) != perRootBytes {
		t.Fatalf("first root returned %d bytes, want %d", len(first), perRootBytes)
	}
	second, err := shared.ReadFile(pathB)
	if !errors.Is(err, errGovernanceReadViewBudget) || len(second) != 0 {
		t.Fatalf("second root exceeded the shared operation budget without rejection: bytes=%d err=%v", len(second), err)
	}
	if len(viewA.unknown["action-requests"]) != 0 {
		t.Fatalf("second-root overflow contaminated the first root: %+v", viewA.unknown)
	}

	op := viewA.operation
	if op == nil {
		t.Fatal("production read-view initializer omitted its operation")
	}
	audit := op.audit()
	metadata, _ := viewA.metadata(projectA, "revision-a", "legacy", audit)
	if metadata.InputBytes > governanceReadViewMaxInputBytes {
		t.Fatalf("operation read view accounted %d bytes above its %d-byte limit", metadata.InputBytes, governanceReadViewMaxInputBytes)
	}
	if metadata.Freshness != ReadViewFreshnessUnknown {
		t.Fatalf("budget exhaustion did not make the operation proof unknown: %+v", metadata)
	}
	var viewB *governanceReadView
	for _, root := range op.roots {
		if root.project == projectB {
			viewB = root.view
			break
		}
	}
	if viewB == nil {
		t.Fatal("production read-view routing did not create the touched second root")
	}
	if len(viewB.unknown["action-requests"]) == 0 {
		t.Fatalf("budget exhaustion was not attributed to the producer reading the second root: %+v", viewB.unknown)
	}
}

func TestGovernanceReadViewDependencyDepthLimitLocalizesCoverage(t *testing.T) {
	governanceReadViewProbeUseKnownEngineVersion(t)
	store := obligationFixture(t)
	writeChain := func(prefix string, edges int) string {
		t.Helper()
		for depth := 0; depth <= edges; depth++ {
			slug := fmt.Sprintf("%s-%02d", prefix, depth)
			status := "done"
			if depth == 0 {
				status = "draft"
			}
			frontmatter := fmt.Sprintf("---\nslug: %s\nstatus: %s\n", slug, status)
			if depth < edges {
				frontmatter += fmt.Sprintf("depends_on: %s-%02d\n", prefix, depth+1)
			}
			frontmatter += "---\n\n# " + slug + "\n"
			path := filepath.Join(store.Root, ".pose", "specs", fmt.Sprintf("2026-10-10-%s-%02d.md", prefix, depth))
			if err := os.WriteFile(path, []byte(frontmatter), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return prefix + "-00"
	}
	// Readiness validates from the first dependency (depth zero): 66 edges
	// from the consumer exceed the legacy 64-depth bound by one. A local
	// chain longer than B34's federated depth eight must remain current.
	deepRoot := writeChain("deepchain", 66)
	shortRoot := writeChain("shortchain", 12)

	short, err := store.ProjectObligations(ObligationQuery{Scope: "spec:" + shortRoot})
	if err != nil {
		t.Fatal(err)
	}
	if short.Snapshot.ReadView == nil || short.Snapshot.ReadView.Freshness != ReadViewFreshnessCurrent {
		t.Fatalf("dependency chain within the depth budget was not current: %+v", short.Snapshot.ReadView)
	}
	if state := governanceReadViewIntegrationCoverage(short, "readiness"); state != CoverageStateCurrent {
		t.Fatalf("short-chain readiness coverage = %q, want current", state)
	}
	if state := governanceReadViewIntegrationCoverage(short, "start"); state != CoverageStateUnsupported {
		t.Fatalf("unadopted start coverage = %q, want unsupported", state)
	}
	if short.Complete {
		t.Fatalf("projection ignored unsupported start coverage and reported complete: %+v", short.Coverage)
	}

	deep, err := store.ProjectObligations(ObligationQuery{Scope: "spec:" + deepRoot})
	if err != nil {
		t.Fatal(err)
	}
	if deep.Snapshot.ReadView == nil || deep.Snapshot.ReadView.Freshness != ReadViewFreshnessUnknown {
		t.Fatalf("dependency-depth exhaustion was reported as a current read view: %+v", deep.Snapshot.ReadView)
	}
	if state := governanceReadViewIntegrationCoverage(deep, "readiness"); state != CoverageStateIncomplete {
		t.Fatalf("depth-limited readiness coverage = %q, want incomplete", state)
	}
	if deep.Complete {
		t.Fatalf("depth-limited projection was reported complete: %+v", deep.Coverage)
	}
	foundLimit := false
	for _, obligation := range deep.Obligations {
		foundLimit = foundLimit || obligation.Source.Producer == "readiness" && strings.Contains(obligation.Message, "resolution-limit")
	}
	if !foundLimit {
		t.Fatalf("depth-limited dependency blocker was lost from the useful result: %+v", deep.Obligations)
	}
	for _, coverage := range deep.Coverage {
		if coverage.Producer == "readiness" {
			continue
		}
		if other := governanceReadViewIntegrationCoverage(short, coverage.Producer); coverage.State != other {
			t.Fatalf("depth limit degraded unrelated %q coverage: deep=%q short=%q", coverage.Producer, coverage.State, other)
		}
	}
}

func governanceReadViewIntegrationCommittedRoot(t *testing.T, projectID string) string {
	t.Helper()
	root := t.TempDir()
	poseDir := filepath.Join(root, ".pose")
	if err := os.MkdirAll(poseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	project := []byte(`{"schema_version":1,"project_id":"` + projectID + `","name":"budget fixture"}` + "\n")
	if err := os.WriteFile(filepath.Join(poseDir, "project.json"), project, 0o644); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) {
		t.Helper()
		commandArgs := append([]string{"-C", root, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid"}, args...)
		if output, err := exec.Command("git", commandArgs...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	git("init", "-q")
	git("add", ".")
	git("commit", "-qm", "governance read-view budget fixture")
	return root
}

func governanceReadViewIntegrationObligation(project, generation string) Obligation {
	o := newObligation(project, "action-requests", specNode(project, "consumer"), "action-request", generation)
	o.Category = ObligationActorAction
	o.ReasonCode = "action-request-generation"
	o.Condition = "the captured generation is awaiting review"
	o.Waiting = WaitingActor
	o.Effects = []ObligationEffect{{Phase: PhaseReview, Mode: EffectAdvisory}}
	o.Message = "captured generation " + generation
	return o
}

func governanceReadViewIntegrationProjectAsync(store Store) <-chan governanceReadViewIntegrationResult {
	result := make(chan governanceReadViewIntegrationResult, 1)
	go func() {
		report, err := store.ProjectObligations(ObligationQuery{})
		result <- governanceReadViewIntegrationResult{report: report, err: err}
	}()
	return result
}

func governanceReadViewIntegrationAwaitCapture(t *testing.T, captured <-chan governanceReadViewIntegrationCapture) governanceReadViewIntegrationCapture {
	t.Helper()
	select {
	case event := <-captured:
		return event
	case <-time.After(20 * time.Second):
		t.Fatal("timed out waiting for the producer's captured generation")
		return governanceReadViewIntegrationCapture{}
	}
}

func governanceReadViewIntegrationAwaitResult(t *testing.T, result <-chan governanceReadViewIntegrationResult) governanceReadViewIntegrationResult {
	t.Helper()
	select {
	case completed := <-result:
		return completed
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for the projection result")
		return governanceReadViewIntegrationResult{}
	}
}

func governanceReadViewIntegrationWriteGeneration(t *testing.T, path, value string, modTime time.Time) {
	t.Helper()
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
	if !modTime.IsZero() {
		if err := os.Chtimes(path, modTime, modTime); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() != 1 || !info.ModTime().Equal(modTime) {
			t.Fatalf("generation mutation did not preserve same-size/mtime identity: size=%d mtime=%v want=%v", info.Size(), info.ModTime(), modTime)
		}
	}
}

func governanceReadViewIntegrationCoverage(report ObligationReport, producer string) string {
	for _, coverage := range report.Coverage {
		if coverage.Producer == producer {
			return coverage.State
		}
	}
	return "missing"
}

func governanceReadViewIntegrationObligationsFrom(report ObligationReport, producer string) []Obligation {
	items := []Obligation{}
	for _, obligation := range report.Obligations {
		if obligation.Source.Producer == producer {
			items = append(items, obligation)
		}
	}
	return items
}

func TestGovernanceReadViewOperationSharesRecordCapacityAndAuditsAcrossRoots(t *testing.T) {
	governanceReadViewProbeUseKnownEngineVersion(t)
	const projectA, projectB = "proj.capacity-a", "proj.capacity-b"
	rootA := governanceReadViewIntegrationCommittedRoot(t, projectA)
	rootB := governanceReadViewIntegrationCommittedRoot(t, projectB)
	seed := filepath.Join(rootA, "seed")
	if err := os.WriteFile(seed, []byte("tiny input"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(seed)
	if err != nil {
		t.Fatal(err)
	}
	dirA, dirB := filepath.Join(rootA, "listing"), filepath.Join(rootB, "listing")
	for _, dir := range []string{dirA, dirB} {
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	files := &governanceReadViewCapacityDirectoryFiles{Files: NativeFiles(), directories: map[string][]fs.DirEntry{
		dirA: governanceReadViewCapacityEntries(info, 16000),
		dirB: governanceReadViewCapacityEntries(info, 16000),
	}}
	resolver := ArtifactResolver{Roots: NewRootsIn(files, RootsConfig{
		DefaultRoot: rootA, DefaultProjectID: projectA, Explicit: map[string]string{projectB: rootB},
	}), FS: files}
	store := Store{Root: rootA, FS: files, FederatedProjectID: projectA, FederatedResolver: &resolver}
	_, view := store.beginGovernanceReadView()
	captured := view.filesForProject("capacity", projectA)
	for _, dir := range []string{dirA, dirB} {
		entries, err := captured.ReadDir(dir)
		if err != nil || len(entries) != 16000 {
			t.Fatalf("new capacity rejected a complete root listing: entries=%d err=%v", len(entries), err)
		}
	}
	beforeReplay := files.opens
	if entries, err := view.filesForProject("consumer", projectA).ReadDir(dirB); err != nil || len(entries) != 16000 || files.opens != beforeReplay {
		t.Fatalf("listing was not replayed from capture: entries=%d opens=%d/%d err=%v", len(entries), files.opens, beforeReplay, err)
	}
	audit := view.audit()
	if governanceReviewAuditChanged(audit) || len(audit.UnknownProducers) != 0 || files.opens <= beforeReplay {
		t.Fatalf("shared final audit did not reobserve complete listings: %+v opens=%d", audit, files.opens)
	}
	metadata, proof := view.metadata(projectA, "revision", "legacy", audit)
	if metadata.Freshness != ReadViewFreshnessCurrent {
		t.Fatalf("new capacity was not certified current: %+v", metadata)
	}
	if changed := proof.changes(files); len(changed) != 0 {
		t.Fatalf("shared proof replay rejected complete listings: %v", changed)
	}
	// Both roots fit separately even below the new cap. Their combined grown
	// listings exceed the 40,000 cap, so shared audit/replay must refuse the closure.
	files.directories[dirA] = governanceReadViewCapacityEntries(info, 20800)
	files.directories[dirB] = governanceReadViewCapacityEntries(info, 20800)
	changed := proof.changes(files)
	unverified := false
	for _, reason := range changed {
		unverified = unverified || strings.HasPrefix(reason, "read-view-unverified:")
	}
	if !unverified {
		t.Fatalf("shared proof replay did not refuse aggregate record overflow: %v", changed)
	}
	_, overflow := store.beginGovernanceReadView()
	for i, dir := range []string{dirA, dirB} {
		entries, err := overflow.filesForProject("overflow", projectA).ReadDir(dir)
		if i == 0 && (err != nil || len(entries) != 20800) {
			t.Fatalf("first root no longer fits independently: entries=%d err=%v", len(entries), err)
		}
		if i == 1 && (!errors.Is(err, errGovernanceReadViewBudget) || len(entries) != 0) {
			t.Fatalf("aggregate record overflow was accepted or truncated: entries=%d err=%v", len(entries), err)
		}
	}
	if audit := overflow.audit(); len(audit.UnknownProducers["overflow"]) == 0 {
		t.Fatalf("aggregate record overflow lost its audit cause: %+v", audit)
	}
}
