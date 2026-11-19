package cli

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

const attributionNamespace = "pose-retroactive-attribution"

type attributionManifest struct {
	SchemaVersion int                 `json:"schema_version"`
	Project       string              `json:"project"`
	Signer        string              `json:"signer"`
	Baseline      string              `json:"baseline"`
	Records       []attributionRecord `json:"records"`
}

type attributionRecord struct {
	Spec     string                 `json:"spec"`
	Base     string                 `json:"base"`
	Head     string                 `json:"head"`
	Witness  string                 `json:"witness"`
	Observed posemodel.ObservedPath `json:"observed"`
	Reason   string                 `json:"reason"`
}

type attributionKey struct {
	spec, base, head string
	observed         posemodel.ObservedPath
}

func readAttributionFile(files posemodel.Files, root, name string, limit int64) ([]byte, error) {
	if err := posemodel.ValidateArtifactPathIn(files, root, name, false); err != nil {
		return nil, err
	}
	path := filepath.Join(root, filepath.FromSlash(name))
	info, err := files.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, fmt.Errorf("%s must be a bounded regular file", name)
	}
	f, err := files.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > limit {
		return nil, fmt.Errorf("%s exceeds its size limit", name)
	}
	return raw, nil
}

// applyArtifactAttributions changes the disposition of one exact finding.
// Keep claims, observed changes and the sealed provenance digest intact.
func applyArtifactAttributions(files posemodel.Files, root string, specs []posemodel.Spec, graph *posemodel.DeliveryIntegrityGraph) error {
	name := posemodel.ArtifactAttributionManifest
	raw, err := readAttributionFile(files, root, name, 128*1024)
	if errors.Is(err, os.ErrNotExist) {
		absent, proofErr := posemodel.ProvenAbsentIn(files, filepath.Join(root, name))
		if proofErr == nil && absent {
			absent, proofErr = posemodel.ProvenAbsentIn(files, filepath.Join(root, name+".sig"))
			if proofErr == nil && absent {
				return nil
			}
			return fmt.Errorf("retroactive attribution: manifest missing beside signature")
		}
	}
	if err != nil {
		return fmt.Errorf("retroactive attribution: read manifest: %w", err)
	}
	sig, err := readAttributionFile(files, root, name+".sig", 16*1024)
	if err != nil {
		return fmt.Errorf("retroactive attribution: read signature: %w", err)
	}
	var manifest attributionManifest
	if err := posemodel.DecodeStrictJSON(raw, &manifest); err != nil {
		return fmt.Errorf("retroactive attribution: invalid manifest: %w", err)
	}
	project, err := (posemodel.Store{Root: root, FS: files}).WriteProjectID()
	if err != nil || manifest.SchemaVersion != 1 || manifest.Project != project || !fullAttributionCommit(manifest.Baseline) || len(manifest.Records) == 0 || len(manifest.Records) > 128 {
		return fmt.Errorf("retroactive attribution: invalid project, baseline, schema or record count")
	}
	policy, err := posemodel.LoadActionPolicyIn(files, root)
	if err != nil {
		return fmt.Errorf("retroactive attribution: read maintainer policy: %w", err)
	}
	if !strings.HasPrefix(manifest.Signer, "human:") || !slices.Contains(policy.RolesOf(manifest.Signer), "maintainer") {
		return fmt.Errorf("retroactive attribution: signer must be an explicit human maintainer")
	}
	signature, err := posemodel.VerifySSHSignature(string(sig), attributionNamespace, raw)
	if err != nil {
		return fmt.Errorf("retroactive attribution: %w", err)
	}
	owner, registered := policy.KeyOwner(signature.Key.Fingerprint())
	if !registered || owner != manifest.Signer {
		return fmt.Errorf("retroactive attribution: signing key is not registered to the maintainer")
	}
	records, err := verifyAttributionHistory(files, root, specs, manifest)
	if err != nil {
		return err
	}
	sets := make(map[string]posemodel.ChangeSet, len(graph.ChangeSets))
	for _, set := range graph.ChangeSets {
		sets[set.ID] = set
	}
	for i, finding := range graph.Findings {
		if finding.Code != "undeclared" {
			continue
		}
		set := sets[finding.ChangeSet]
		var match posemodel.ObservedPath
		count := 0
		for _, observed := range set.Paths {
			if firstObservedCLIPath(observed) == finding.Path {
				match = observed
				count++
			}
		}
		// The graph finding stores the first path. An ambiguous first path
		// cannot identify the exact action the maintainer authorized.
		key := attributionKey{set.Spec, set.ResolvedBase, set.ResolvedHead, match}
		if record, ok := records[key]; ok && count == 1 {
			graph.Findings[i] = posemodel.NewDeliveryIntegrityFinding(
				"retroactive-attribution", "info", finding.Spec, finding.Path, finding.ChangeSet,
				fmt.Sprintf("historical attribution signed by %s (%s), witness %s: %s", manifest.Signer, signature.Key.Fingerprint(), record.Witness, record.Reason),
				"see "+name,
			)
		}
	}
	sort.Slice(graph.Findings, func(i, j int) bool { return graph.Findings[i].ID < graph.Findings[j].ID })
	graph.InputDigest = fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(fmt.Sprintf("%s\x00%x\x00%x", graph.InputDigest, sha256.Sum256(raw), sha256.Sum256(sig)))))
	return nil
}

func verifyAttributionHistory(files posemodel.Files, root string, specs []posemodel.Spec, manifest attributionManifest) (map[attributionKey]attributionRecord, error) {
	if _, err := gitOutputBoundedIn(files, root, 4096, "merge-base", "--is-ancestor", manifest.Baseline, "HEAD"); err != nil {
		return nil, fmt.Errorf("retroactive attribution: baseline is outside current Git history: %w", err)
	}
	known := make(map[string]bool, len(specs))
	for _, spec := range specs {
		known[spec.Slug] = spec.Status == "done"
	}
	records := make(map[attributionKey]attributionRecord, len(manifest.Records))
	ranges := map[[2]string][]string{}
	witnesses := map[string][]posemodel.ObservedPath{}
	for _, record := range manifest.Records {
		key := attributionKey{record.Spec, record.Base, record.Head, record.Observed}
		_, duplicate := records[key]
		if !known[record.Spec] || duplicate || !fullAttributionCommit(record.Base) || !fullAttributionCommit(record.Head) || !fullAttributionCommit(record.Witness) ||
			strings.TrimSpace(record.Reason) == "" || len(record.Reason) > 1024 || strings.ContainsAny(record.Reason, "\r\n") {
			return nil, fmt.Errorf("retroactive attribution: invalid or duplicate record for spec %q", record.Spec)
		}
		bounds := [2]string{record.Base, record.Head}
		commits, present := ranges[bounds]
		if !present {
			if _, err := gitOutputBoundedIn(files, root, 4096, "merge-base", "--is-ancestor", record.Head, manifest.Baseline); err != nil {
				return nil, fmt.Errorf("retroactive attribution: range is outside baseline: %w", err)
			}
			raw, err := gitOutputBoundedIn(files, root, 4*1024*1024, "rev-list", posemodel.ChangeSetRange(record.Base, record.Head), "--")
			if err != nil {
				return nil, fmt.Errorf("retroactive attribution: read historical range: %w", err)
			}
			commits = strings.Fields(string(raw))
			ranges[bounds] = commits
		}
		if !slices.Contains(commits, record.Witness) {
			return nil, fmt.Errorf("retroactive attribution: witness is outside range for %s", record.Spec)
		}
		paths, present := witnesses[record.Witness]
		if !present {
			raw, err := gitOutputBoundedIn(files, root, 4*1024*1024, "diff-tree", "--root", "-m", "--no-commit-id", "--name-status", "-z", "-M", "-r", record.Witness, "--")
			if err != nil {
				return nil, fmt.Errorf("retroactive attribution: read witness: %w", err)
			}
			paths, err = parseGitNameStatusNUL(raw)
			if err != nil {
				return nil, fmt.Errorf("retroactive attribution: invalid witness diff: %w", err)
			}
			witnesses[record.Witness] = paths
		}
		if !slices.Contains(paths, record.Observed) {
			return nil, fmt.Errorf("retroactive attribution: witness did not make the exact action and path for %s", record.Spec)
		}
		records[key] = record
	}
	return records, nil
}

func fullAttributionCommit(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	for _, c := range value {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
