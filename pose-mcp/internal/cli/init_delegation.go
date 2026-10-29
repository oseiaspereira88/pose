package cli

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

var delegationLookupCLI = exec.LookPath

// A fresh init offers this separately from the validation-matrix wizard.
// Discovery looks up executables; it never launches a vendor or probes quota.
func initDelegation(root string, stdout, stderr io.Writer) int {
	if err := posemodel.ValidateDelegationPaths(root); err != nil {
		return initDelegationError(stderr, err)
	}
	docs, err := posemodel.LoadPolicyDocs(root)
	if err != nil {
		return initDelegationError(stderr, err)
	}
	input, interactive := setupInput()
	accept := false
	if interactive {
		render(stdout, stderr).ContractLine(tr("Enable delegated review and pin this machine’s native executor signing key? [Y/n] ", "Ativar review delegado e registrar a chave de assinatura do executor desta máquina? [S/n] "))
		line, readErr := bufio.NewReader(input).ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return initDelegationError(stderr, readErr)
		}
		// EOF without an answer is not consent; an entered empty line is yes.
		if readErr == nil || strings.TrimSpace(line) != "" {
			switch strings.ToLower(strings.TrimSpace(line)) {
			case "", "y", "yes", "s", "sim":
				accept = true
			case "n", "no", "nao", "não":
			default:
				return initDelegationError(stderr, fmt.Errorf("answer yes or no; delegation remains disabled"))
			}
		}
	}
	policy := posemodel.ExampleDelegationPolicy()
	for name, agent := range policy.Agents {
		if _, err := delegationLookupCLI(agent.CLI); err != nil {
			delete(policy.Agents, name)
		}
	}
	if accept && len(policy.Agents) == 0 {
		render(stdout, stderr).Hint(tr("[INFO] no supported agent CLI found; delegation stays off. Configure delegation.json, then pose adopt delegated-review.", "[INFO] nenhuma CLI de agente suportada encontrada; delegação continua desligada. Configure delegation.json e execute pose adopt delegated-review."))
		accept = false
	}
	if !accept {
		docs.Review["delegated_review"] = false
		docs.Review["delegated_review_disabled_at"] = time.Now().UTC().Format(time.DateOnly)
		if err := docs.Write(root, posemodel.Store{Root: root}); err != nil {
			return initDelegationError(stderr, err)
		}
		return 0
	}
	// Never replace a pre-existing policy, including a dangling link or a
	// legacy policy awaiting migration. A fresh init must not erase user data.
	for _, name := range []string{"delegation.json", "reviewers.json"} {
		if _, err := os.Lstat(filepath.Join(root, ".pose", "policy", name)); err == nil || !os.IsNotExist(err) {
			return initDelegationError(stderr, fmt.Errorf("existing %s is preserved; configure it and use pose adopt delegated-review", name))
		}
	}
	project, found, err := posemodel.ReadProjectFileIn(posemodel.NativeFiles(), root)
	if err != nil || !found {
		return initDelegationError(stderr, fmt.Errorf("a declared project identity is required: %v", err))
	}
	audience, _ := docs.Review["authority_audience"].(string)
	if audience == "" {
		audience = project
	}
	if prior, _ := docs.Review["authority_project"].(string); prior != "" && prior != project {
		return initDelegationError(stderr, fmt.Errorf("authority_project differs from this project"))
	}
	sum := sha256.Sum256([]byte(project))
	name := fmt.Sprintf("delegation-%x", sum[:8])
	key, err := posemodel.LoadIssuerKey(root, name)
	if err != nil {
		loadErr := err
		key, err = posemodel.CreateIssuerKey(root, name, time.Now())
		if err != nil {
			return initDelegationError(stderr, fmt.Errorf("load executor issuer: %v; create executor issuer: %w", loadErr, err))
		}
	}
	if err != nil {
		return initDelegationError(stderr, err)
	}
	if _, err := posemodel.PlanIssuerPin(docs, key.Pin(), true, false, audience, project); err != nil {
		return initDelegationError(stderr, err)
	}
	policy.Provenance = &posemodel.ReviewRunProvenancePolicy{Trust: "executor-attested"}
	policy.Conclusions = &posemodel.DelegationConclusions{Issuer: name}
	for id, agent := range policy.Agents {
		agent.Executor = "agent:" + id
		policy.Agents[id] = agent
	}
	raw, err := posemodel.CanonicalDelegationPolicy(policy)
	if err != nil {
		return initDelegationError(stderr, err)
	}
	docs.Review["delegated_review"] = true
	docs.Review["delegated_review_adopted_at"] = time.Now().UTC().Format(time.DateOnly)
	if _, _, _, _, err := docs.Rendered(posemodel.Store{Root: root}); err != nil {
		return initDelegationError(stderr, err)
	}
	// The decision is published last. A conflict never leaves an enabled flag
	// without a policy. On a later failure preserve the policy/key for recovery,
	// rather than deleting a file another writer might already have changed.
	if err := posemodel.PublishFile(filepath.Join(root, ".pose", "policy", "delegation.json"), raw, 0o600, posemodel.PublishAuthoritative, posemodel.PublishExpectation{Mode: posemodel.ExpectAbsent}); err != nil {
		return initDelegationError(stderr, err)
	}
	if err := docs.Write(root, posemodel.Store{Root: root}); err != nil {
		return initDelegationError(stderr, fmt.Errorf("delegation policy was saved but activation failed; preserve it, pin issuer %s with pose issuer pin --attestations and use pose adopt delegated-review: %w", name, err))
	}
	render(stdout, stderr).Hint(tr("[INFO] delegated review enabled; native executor signatures are declarations, not vendor receipts or human approval.", "[INFO] review delegado ativado; assinaturas do executor nativo são declarações, não recibos do fornecedor nem aprovação humana."))
	return 0
}

func initDelegationError(stderr io.Writer, err error) int {
	render(io.Discard, stderr).Failure("pose init: " + err.Error())
	return 1
}

func suggestDelegationUpdate(files posemodel.Files, root string, stdout io.Writer) {
	docs, err := posemodel.LoadPolicyDocsIn(files, root)
	if err != nil {
		return
	}
	if on, _ := docs.Review["delegated_review"].(bool); !on {
		render(stdout, stdout).Hint(tr("[INFO] delegated review is off; configure agents and run pose adopt delegated-review --apply when ready.", "[INFO] review delegado está desligado; configure os agentes e execute pose adopt delegated-review --apply quando estiver pronto."))
	}
}
