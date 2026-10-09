package cli

// pose issuer — native attestation issuers (spec pose-native-attestation-issuer).
// A key lives outside the project; policy trusts it through a pin. Nothing
// here prints or returns private key material.

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

const issuerUsage = "Usage: pose issuer init <name> | pose issuer pin <name|name:public-key> [--attestations] [--human-authority] [--audience <id>] [--apply] | pose issuer list [--json] | pose issuer rotate <name> [--apply]"

func cmdIssuer(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, issuerUsage)
	}
	switch args[0] {
	case "init":
		return cmdIssuerInit(args[1:], stdout, stderr)
	case "pin":
		return cmdIssuerPin(root, args[1:], stdout, stderr)
	case "list":
		return cmdIssuerList(root, args[1:], stdout, stderr)
	case "rotate":
		return cmdIssuerRotate(root, args[1:], stdout, stderr)
	}
	return usageError(stderr, issuerUsage)
}

func cmdIssuerInit(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 || strings.HasPrefix(args[0], "-") {
		return usageError(stderr, issuerUsage)
	}
	key, err := posemodel.CreateIssuerKey(args[0], time.Now())
	if err != nil {
		render(stdout, stderr).Failure("pose issuer init: " + strings.TrimPrefix(err.Error(), "pose: "))
		return 1
	}
	out := render(stdout, stderr)
	out.Field("issuer.name", key.Name)
	out.Field("issuer.pin", key.Pin())
	out.Field("issuer.public_key", key.PublicKey())
	out.Field("issuer.key_path", key.Path)
	out.Field("issuer.next", "pose issuer pin "+key.Name+" --attestations --human-authority --apply")
	return 0
}

// resolveIssuerPin accepts a local issuer name, or `name:<base64 public key>`
// for an issuer whose key lives elsewhere (another machine, a teammate).
func resolveIssuerPin(value string) (string, error) {
	if name, public, ok := strings.Cut(value, ":"); ok {
		if err := posemodel.ValidIssuerName(name); err != nil {
			return "", err
		}
		raw, err := base64.StdEncoding.DecodeString(public)
		if err != nil || len(raw) != ed25519.PublicKeySize {
			return "", fmt.Errorf("%q is not a base64 Ed25519 public key", public)
		}
		return posemodel.IssuerPin(name, raw), nil
	}
	key, err := posemodel.LoadIssuerKey(value)
	if err != nil {
		return "", err
	}
	return key.Pin(), nil
}

func cmdIssuerPin(root string, args []string, stdout, stderr io.Writer) int {
	var target, audience string
	attestations, humanAuthority, apply := false, false, false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--attestations":
			attestations = true
		case "--human-authority":
			humanAuthority = true
		case "--apply":
			apply = true
		case "--audience":
			if i+1 >= len(args) {
				return usageError(stderr, issuerUsage)
			}
			i++
			audience = args[i]
		default:
			if strings.HasPrefix(args[i], "-") || target != "" {
				return usageError(stderr, issuerUsage)
			}
			target = args[i]
		}
	}
	if target == "" {
		return usageError(stderr, issuerUsage)
	}
	if !attestations && !humanAuthority {
		attestations = true
	}
	return pinIssuer(root, target, attestations, humanAuthority, audience, apply, "pose issuer pin", stdout, stderr)
}

func pinIssuer(root, target string, attestations, humanAuthority bool, audience string, apply bool, command string, stdout, stderr io.Writer) int {
	pin, err := resolveIssuerPin(target)
	if err != nil {
		render(stdout, stderr).Failure(command + ": " + strings.TrimPrefix(err.Error(), "pose: "))
		return 1
	}
	docs, err := posemodel.LoadPolicyDocs(root)
	if err != nil {
		render(stdout, stderr).Failure(command + ": " + err.Error())
		return 1
	}
	project := ""
	if humanAuthority {
		// verified identity needs to know which project a claim governs.
		if id, ok, _ := posemodel.ReadProjectFile(root); ok {
			project = id
		}
		if current, _ := docs.Review["authority_audience"].(string); current == "" && audience == "" {
			audience = project
		}
	}
	plan, err := posemodel.PlanIssuerPin(docs, pin, attestations, humanAuthority, audience, project)
	if err != nil {
		render(stdout, stderr).Failure(command + ": " + strings.TrimPrefix(err.Error(), "pose: "))
		return 1
	}
	render(stdout, stderr).Field("issuer.pin", pin)
	if len(plan.Changes) == 0 {
		render(stdout, stderr).Field("issuer.result", "already pinned; nothing to change")
		return 0
	}
	for _, change := range plan.Changes {
		render(stdout, stderr).Field("issuer.change", change)
	}
	if !apply {
		render(stdout, stderr).Field("issuer.apply", "false")
		return 0
	}
	if err := docs.Write(root, posemodel.Store{Root: root}); err != nil {
		render(stdout, stderr).Failure(command + ": " + err.Error())
		return 1
	}
	render(stdout, stderr).Field("issuer.apply", "true")
	return 0
}

func cmdIssuerList(root string, args []string, stdout, stderr io.Writer) int {
	asJSON := len(args) == 1 && args[0] == "--json"
	if len(args) > 1 || len(args) == 1 && !asJSON {
		return usageError(stderr, issuerUsage)
	}
	keys, err := posemodel.ListIssuerKeys()
	if err != nil {
		render(stdout, stderr).Failure("pose issuer list: " + err.Error())
		return 1
	}
	docs, err := posemodel.LoadPolicyDocs(root)
	if err != nil {
		render(stdout, stderr).Failure("pose issuer list: " + err.Error())
		return 1
	}
	listed := func(key string) []string {
		out := []string{}
		if items, ok := docs.Review[key].([]any); ok {
			for _, item := range items {
				if s, ok := item.(string); ok {
					out = append(out, s)
				}
			}
		}
		return out
	}
	type pinned struct {
		Pin            string `json:"pin"`
		Attestations   bool   `json:"attestations"`
		HumanAuthority bool   `json:"human_authority"`
		LocalKey       bool   `json:"local_key"`
	}
	attest, human := listed("trusted_attestation_issuers"), listed("human_authority_issuers")
	local := map[string]bool{}
	for _, k := range keys {
		local[k.Pin] = true
	}
	pins := []pinned{}
	seen := map[string]bool{}
	for _, pin := range append(append([]string{}, attest...), human...) {
		if seen[pin] {
			continue
		}
		seen[pin] = true
		p := pinned{Pin: pin, LocalKey: local[pin]}
		for _, a := range attest {
			p.Attestations = p.Attestations || a == pin
		}
		for _, h := range human {
			p.HumanAuthority = p.HumanAuthority || h == pin
		}
		pins = append(pins, p)
	}
	if asJSON {
		raw, _ := json.MarshalIndent(map[string]any{"policy_pins": pins, "local_keys": keys}, "", "  ")
		render(stdout, stderr).ContractLine(string(raw))
		return 0
	}
	if len(pins) == 0 {
		render(stdout, stderr).Field("issuer.policy", "no issuer is pinned")
	}
	for _, p := range pins {
		grants := []string{}
		if p.Attestations {
			grants = append(grants, "attestations")
		}
		if p.HumanAuthority {
			grants = append(grants, "human-authority")
		}
		where := "external"
		if p.LocalKey {
			where = "local key"
		}
		render(stdout, stderr).Field("issuer.pinned", fmt.Sprintf("%s (%s; %s)", p.Pin, strings.Join(grants, ", "), where))
	}
	for _, k := range keys {
		state := "current"
		if k.Retired {
			state = "retired"
		}
		render(stdout, stderr).Field("issuer.key", fmt.Sprintf("%s %s (%s, created %s)", k.Issuer, k.Pin, state, k.CreatedAt))
	}
	return 0
}

func cmdIssuerRotate(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || len(args) > 2 || strings.HasPrefix(args[0], "-") || len(args) == 2 && args[1] != "--apply" {
		return usageError(stderr, issuerUsage)
	}
	name, apply := args[0], len(args) == 2
	old, err := posemodel.LoadIssuerKey(name)
	if err != nil {
		render(stdout, stderr).Failure("pose issuer rotate: " + strings.TrimPrefix(err.Error(), "pose: "))
		return 1
	}
	render(stdout, stderr).Field("issuer.current", old.Pin())
	if !apply {
		render(stdout, stderr).Field("issuer.plan", "retire the current key, create a new one and pin it next to the old pin; the old pin stays until you remove it")
		render(stdout, stderr).Field("issuer.apply", "false")
		return 0
	}
	docs, err := posemodel.LoadPolicyDocs(root)
	if err != nil {
		render(stdout, stderr).Failure("pose issuer rotate: " + err.Error())
		return 1
	}
	inList := func(key string) bool {
		if items, ok := docs.Review[key].([]any); ok {
			for _, item := range items {
				if item == old.Pin() {
					return true
				}
			}
		}
		return false
	}
	attestations, human := inList("trusted_attestation_issuers"), inList("human_authority_issuers")
	_, current, err := posemodel.RotateIssuerKey(name, time.Now())
	if err != nil {
		render(stdout, stderr).Failure("pose issuer rotate: " + strings.TrimPrefix(err.Error(), "pose: "))
		return 1
	}
	render(stdout, stderr).Field("issuer.new", current.Pin())
	if !attestations && !human {
		render(stdout, stderr).Field("issuer.policy", "the old key was not pinned here; pin the new one with pose issuer pin "+name)
		return 0
	}
	return pinIssuer(root, name, attestations, human, "", true, "pose issuer rotate", stdout, stderr)
}
