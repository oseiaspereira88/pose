package cli

// pose identity — register the SSH public keys a principal signs answers
// with (spec pose-signed-action-answers). The git identity only suggests a
// name: anyone can set user.email, so the proof is the key, never the name.

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

const identityUsage = "Usage: pose identity add [<principal>] --key <public-key-file|line> [--role <role>] [--apply] | pose identity list [--json] | pose identity remove <principal> [--fingerprint <SHA256:...>] [--apply]"

func cmdIdentity(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, identityUsage)
	}
	switch args[0] {
	case "add":
		return cmdIdentityAdd(root, args[1:], stdout, stderr)
	case "list":
		return cmdIdentityList(root, args[1:], stdout, stderr)
	case "remove":
		return cmdIdentityRemove(root, args[1:], stdout, stderr)
	}
	return usageError(stderr, identityUsage)
}

var principalNameUnsafe = regexp.MustCompile(`[^a-z0-9._-]+`)

// suggestedPrincipal derives `human:<name>` from the git identity, or "".
func suggestedPrincipal(root string) (principal, source string) {
	read := func(key string) string {
		out, err := exec.Command("git", "-C", root, "config", "--get", key).Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}
	name, from := "", ""
	if email := read("user.email"); email != "" {
		name, from = strings.SplitN(email, "@", 2)[0], "git user.email"
	} else if user := read("user.name"); user != "" {
		name, from = user, "git user.name"
	}
	name = strings.Trim(principalNameUnsafe.ReplaceAllString(strings.ToLower(name), "-"), "-.")
	if name == "" {
		return "", ""
	}
	return "human:" + name, from
}

// readPublicKeyArg accepts a public key file or the key line itself.
func readPublicKeyArg(value string) (posemodel.SSHPublicKey, error) {
	line := value
	if raw, err := os.ReadFile(value); err == nil {
		line = ""
		for _, candidate := range strings.Split(string(raw), "\n") {
			if candidate = strings.TrimSpace(candidate); candidate != "" && !strings.HasPrefix(candidate, "#") {
				line = candidate
				break
			}
		}
		if strings.Contains(string(raw), "PRIVATE KEY") {
			return posemodel.SSHPublicKey{}, fmt.Errorf("%s is a private key; pass the .pub file — POSE never reads a private key", value)
		}
	}
	return posemodel.ParseAuthorizedKey(line)
}

func presenceText(key posemodel.SSHPublicKey) string {
	if key.ProvesPresence() {
		return "proves presence: every signature records whether the security key was touched"
	}
	return "does not prove presence: anyone holding the key file can sign; for a person, prefer a security key (ssh-keygen -t ed25519-sk)"
}

func cmdIdentityAdd(root string, args []string, stdout, stderr io.Writer) int {
	principal, keyArg, role, apply := "", "", "", false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--apply":
			apply = true
		case "--key", "--role":
			if i+1 >= len(args) {
				return usageError(stderr, identityUsage)
			}
			if args[i] == "--key" {
				keyArg = args[i+1]
			} else {
				role = args[i+1]
			}
			i++
		default:
			if strings.HasPrefix(args[i], "--") || principal != "" {
				return usageError(stderr, identityUsage)
			}
			principal = args[i]
		}
	}
	out := render(stdout, stderr)
	if keyArg == "" {
		out.Failure("pose identity add: --key is required: the key is the proof, a name alone proves nothing")
		return 2
	}
	if principal == "" {
		suggestion, source := suggestedPrincipal(root)
		if suggestion == "" {
			out.Failure("pose identity add: name the principal (human:<name> or agent:<name>); no git identity to suggest one")
			return 2
		}
		principal = suggestion
		out.Field("identity.suggested", principal+" (from "+source+"; a name, not a proof — the key is the proof)")
	}
	key, err := readPublicKeyArg(keyArg)
	if err != nil {
		out.Failure("pose identity add: " + err.Error())
		return 2
	}
	doc, err := posemodel.LoadActionPolicyDocument(root)
	if err != nil {
		out.Failure("pose identity add: " + err.Error())
		return 1
	}
	added, err := doc.AddKey(principal, key, key.Comment, time.Now().UTC().Format(time.DateOnly))
	if err != nil {
		out.Failure("pose identity add: " + err.Error())
		return 1
	}
	granted := role != "" && doc.GrantRole(role, principal)
	out.Field("identity.principal", principal)
	out.Field("identity.key", key.Type+" "+key.Fingerprint()+strings.TrimRight(" "+key.Comment, " "))
	out.Field("identity.presence", presenceText(key))
	if !added {
		out.Field("identity.key_state", "already registered to "+principal)
	}
	if role != "" {
		state := "granted"
		if !granted {
			state = "already held"
		}
		out.Field("identity.role", role+" ("+state+")")
	}
	if strings.HasPrefix(principal, "human:") && !key.ProvesPresence() && doc.RequirePresence {
		out.Failure("pose identity add: require_presence is set, and this key cannot sign an answer that policy accepts from " + principal)
		return 1
	}
	out.Field("identity.apply", boolString(apply))
	if !apply || !added && !granted {
		return 0
	}
	if err := doc.Write(root); err != nil {
		out.Failure("pose identity add: " + err.Error())
		return 1
	}
	out.Field("identity.next", "answers by "+principal+" are verified when signed: pose action resolve <act-id> ... --sign <private key>")
	return 0
}

type identityKeyView struct {
	Type        string `json:"type"`
	Fingerprint string `json:"fingerprint"`
	Comment     string `json:"comment,omitempty"`
	Presence    bool   `json:"proves_presence"`
	AddedAt     string `json:"added_at,omitempty"`
}

type identityView struct {
	Principal string            `json:"principal"`
	Roles     []string          `json:"roles"`
	Keys      []identityKeyView `json:"keys"`
}

func cmdIdentityList(root string, args []string, stdout, stderr io.Writer) int {
	jsonOutput := false
	for _, arg := range args {
		if arg != "--json" {
			return usageError(stderr, identityUsage)
		}
		jsonOutput = true
	}
	out := render(stdout, stderr)
	policy, err := posemodel.LoadActionPolicy(root)
	if err != nil {
		out.Failure("pose identity list: " + err.Error())
		return 1
	}
	views := []identityView{}
	for _, principal := range policy.Principals() {
		view := identityView{Principal: principal, Roles: policy.RolesOf(principal), Keys: []identityKeyView{}}
		if view.Roles == nil {
			view.Roles = []string{}
		}
		for _, entry := range policy.Keys[principal] {
			key, err := posemodel.ParseAuthorizedKey(entry.Key)
			if err != nil {
				continue
			}
			view.Keys = append(view.Keys, identityKeyView{Type: key.Type, Fingerprint: key.Fingerprint(), Comment: key.Comment, Presence: key.ProvesPresence(), AddedAt: entry.AddedAt})
		}
		views = append(views, view)
	}
	sort.Slice(views, func(i, j int) bool { return views[i].Principal < views[j].Principal })
	if jsonOutput {
		return writeJSON(stdout, map[string]any{"identity_assurance": policy.IdentityAssurance, "require_presence": policy.RequirePresence, "principals": views})
	}
	out.Field("identity.assurance", policy.IdentityAssurance)
	if len(views) == 0 {
		out.Field("identity.principals", "none — `pose identity add --key <file.pub> --role maintainer --apply` registers you")
	}
	for _, view := range views {
		line := "roles: " + strings.Join(view.Roles, ", ")
		if len(view.Roles) == 0 {
			line = "roles: none"
		}
		if len(view.Keys) == 0 {
			line += "; no key (answers are declared)"
		}
		for _, key := range view.Keys {
			presence := "no presence"
			if key.Presence {
				presence = "presence"
			}
			line += fmt.Sprintf("; %s %s (%s)", key.Type, key.Fingerprint, presence)
		}
		out.Field("identity."+view.Principal, line)
	}
	return 0
}

func cmdIdentityRemove(root string, args []string, stdout, stderr io.Writer) int {
	principal, fingerprint, apply := "", "", false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--apply":
			apply = true
		case "--fingerprint":
			if i+1 >= len(args) {
				return usageError(stderr, identityUsage)
			}
			fingerprint = args[i+1]
			i++
		default:
			if strings.HasPrefix(args[i], "--") || principal != "" {
				return usageError(stderr, identityUsage)
			}
			principal = args[i]
		}
	}
	if principal == "" {
		return usageError(stderr, identityUsage)
	}
	out := render(stdout, stderr)
	doc, err := posemodel.LoadActionPolicyDocument(root)
	if err != nil {
		out.Failure("pose identity remove: " + err.Error())
		return 1
	}
	removed := doc.RemoveKeys(principal, fingerprint)
	if len(removed) == 0 {
		out.Failure("pose identity remove: no matching key registered to " + principal)
		return 1
	}
	for _, fp := range removed {
		out.Field("identity.remove", principal+" "+fp)
	}
	out.Field("identity.note", "roles are unchanged, and answers already recorded as verified stay verified; new answers by this key are refused")
	out.Field("identity.apply", boolString(apply))
	if !apply {
		return 0
	}
	if err := doc.Write(root); err != nil {
		out.Failure("pose identity remove: " + err.Error())
		return 1
	}
	return 0
}
