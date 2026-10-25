package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// cmdActionResolve records an answer, cancellation, waiver or invalidation
// (spec pose-action-request-resolution).
func cmdActionResolve(root, verb string, args []string, stdout, stderr io.Writer) int {
	usage := "Usage: pose action " + verb + " <act-id> --actor <principal> --request-digest <digest> --expected-revision <n> --idempotency-key <key> " +
		"[--answer <option|text>] [--evidence <ref>]... [--reason <text>] [--role <role>] [--execution <id>] [--channel <name>] [--prepared-by <p>] [--applied-by <p>] " +
		"[--confirmation-mode adopted-conclusions|authorized-operation] [--claim <project-relative json>] [--sign <ssh key> | --signature <file>] [--apply] [--json]"
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return usageError(stderr, usage)
	}
	res := posemodel.ActionResolution{RequestID: args[0], ExpectedRevision: -2}
	switch verb {
	case "resolve":
		res.Type = posemodel.ActionEventAnswered
	case "cancel":
		res.Type = posemodel.ActionEventCancelled
	case "waive":
		res.Type = posemodel.ActionEventWaived
	case "invalidate":
		res.Type = posemodel.ActionEventInvalidated
	}
	apply, jsonOutput, claimPath, signKey, signatureFile := false, false, "", "", ""
	for i := 1; i < len(args); i++ {
		flag := args[i]
		switch flag {
		case "--apply":
			apply = true
			continue
		case "--json":
			jsonOutput = true
			continue
		}
		if i+1 >= len(args) {
			return usageError(stderr, usage)
		}
		i++
		value := args[i]
		switch flag {
		case "--actor":
			res.Actor = value
		case "--request-digest":
			res.RequestDigest = value
		case "--expected-revision":
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 {
				return usageError(stderr, "--expected-revision takes the revision shown by `pose action show`")
			}
			res.ExpectedRevision = n
		case "--idempotency-key":
			res.IdempotencyKey = value
		case "--answer":
			res.Answer = value
		case "--evidence":
			res.Evidence = append(res.Evidence, value)
		case "--reason":
			res.Reason = value
		case "--role":
			res.Role = value
		case "--execution":
			res.Execution = value
		case "--channel":
			res.Channel = value
		case "--prepared-by":
			res.PreparedBy = value
		case "--applied-by":
			res.AppliedBy = value
		case "--confirmation-mode":
			res.ConfirmationMode = value
		case "--claim":
			claimPath = value
		case "--sign":
			signKey = value
		case "--signature":
			signatureFile = value
		default:
			return usageError(stderr, usage)
		}
	}
	if res.ExpectedRevision == -2 {
		return usageError(stderr, "--expected-revision is required: a write never lands on a revision the writer did not read")
	}
	if claimPath != "" {
		if err := posemodel.ValidateArtifactPath(root, claimPath, false); err != nil {
			render(stdout, stderr).Failure("pose action " + verb + ": " + err.Error())
			return 2
		}
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(claimPath)))
		if err != nil {
			render(stdout, stderr).Failure("pose action " + verb + ": " + err.Error())
			return 2
		}
		var signed struct {
			Claim    posemodel.ActionAuthorityClaim `json:"claim"`
			Envelope posemodel.ActionClaimEnvelope  `json:"envelope"`
		}
		if err := json.Unmarshal(raw, &signed); err != nil {
			render(stdout, stderr).Failure("pose action " + verb + ": invalid claim file: " + err.Error())
			return 2
		}
		res.Claim, res.Envelope = &signed.Claim, &signed.Envelope
	}
	out := render(stdout, stderr)
	store := posemodel.Store{Root: root}
	var presencePolicy posemodel.ActionPresencePolicyView
	if res.Type == posemodel.ActionEventAnswered {
		policy, err := posemodel.LoadActionPolicy(root)
		if err != nil {
			out.Failure("pose action " + verb + ": " + err.Error())
			return 1
		}
		presencePolicy = policy.PresencePolicy()
		if apply {
			if err := policy.CheckPresenceMigration(res.Actor); err != nil {
				out.Failure("pose action " + verb + ": " + err.Error())
				return 1
			}
		}
	}
	if apply && signKey != "" {
		if _, err := store.PreflightActionResolution(res, time.Now()); err != nil {
			out.Failure("pose action " + verb + ": " + err.Error())
			return 1
		}
	}
	if signKey != "" || signatureFile != "" {
		if signKey != "" && signatureFile != "" {
			return usageError(stderr, "--sign or --signature (one of them) proves an answer: use it with pose action resolve")
		}
		if signatureFile != "" {
			raw, err := os.ReadFile(signatureFile)
			if err != nil {
				out.Failure("pose action " + verb + ": " + err.Error())
				return 2
			}
			res.Signature = string(raw)
		} else if apply {
			// Signing happens only when the answer is recorded, so a preview
			// never asks for a touch.
			view, err := store.LoadActionRequest(res.RequestID)
			if err != nil {
				out.Failure("pose action " + verb + ": " + err.Error())
				return 1
			}
			statement := posemodel.AnswerStatement(view.Request, res.Actor, res.Answer, res.IdempotencyKey, res.Reason).Canonical()
			namespace := posemodel.ActionAnswerNamespace
			if res.Type != posemodel.ActionEventAnswered {
				statement = posemodel.LifecycleStatement(view.Request, res).Canonical()
				namespace = posemodel.ActionLifecycleNamespace
			}
			signature, err := sshSignNamespace(signKey, statement, namespace)
			if err != nil {
				out.Failure("pose action " + verb + ": " + err.Error())
				return 1
			}
			res.Signature = signature
		}
	}
	if !apply {
		preflight, err := store.PreflightActionResolution(res, time.Now())
		if jsonOutput {
			body := map[string]any{"preview": true, "request": preflight.Request, "admissible": preflight.Admissible, "acceptance": preflight.Acceptance, "proof_pending": preflight.ProofPending, "audit_persisted": false, "refusal": preflight.Refusal, "presence_policy": presencePolicy, "presence_evaluation": "not-evaluated"}
			if res.Type != posemodel.ActionEventAnswered {
				body["operation"] = res.Type
				body["statement"] = string(posemodel.LifecycleStatement(preflight.Request.Request, res).Canonical())
				body["namespace"] = posemodel.ActionLifecycleNamespace
			}
			code := writeJSON(stdout, body)
			if err != nil {
				return 1
			}
			return code
		}
		out.Field("action.preview", "true")
		out.Field("action.acceptance", "not-certified")
		out.Field("action.proof_pending", fmt.Sprint(preflight.ProofPending))
		out.Field("action.audit_persisted", "false")
		if err != nil {
			out.Failure("pose action " + verb + ": " + err.Error())
			return 1
		}
		if res.Type == posemodel.ActionEventAnswered {
			out.Field("action.presence_policy.scope", presencePolicy.Scope)
			out.Field("action.presence_policy.evaluation", presencePolicy.Evaluation)
		}
		renderActionView(out, preflight.Request)
		out.Field("action.apply", "false (apply revalidates the current request, authority and proof)")
		return 0
	}

	view, err := store.ResolveActionRequest(res, time.Now())
	if err != nil {
		out.Failure("pose action " + verb + ": " + err.Error())
		return 1
	}
	if jsonOutput {
		return writeJSON(stdout, view)
	}
	renderActionView(out, view)
	return 0
}

// sshSign signs message with the user's own ssh-keygen; POSE hands it the key
// path and never reads the key. A security key asks for its touch here.
func sshSign(keyPath string, message []byte) (string, error) {
	return sshSignNamespace(keyPath, message, posemodel.ActionAnswerNamespace)
}
func sshSignNamespace(keyPath string, message []byte, namespace string) (string, error) {
	keygen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		return "", errors.New("ssh-keygen is not installed; sign `pose action statement` elsewhere and pass --signature <file>")
	}
	cmd := exec.Command(keygen, "-Y", "sign", "-f", keyPath, "-n", namespace)
	cmd.Stdin = bytes.NewReader(message)
	var signature, diagnostics bytes.Buffer
	cmd.Stdout, cmd.Stderr = &signature, &diagnostics
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ssh-keygen -Y sign failed: %v: %s", err, strings.TrimSpace(diagnostics.String()))
	}
	return signature.String(), nil
}

// cmdActionStatement prints the exact bytes an answer signature covers, for
// a principal who signs on another machine:
// pose action statement <id> --actor <p> --answer <a> --idempotency-key <k> | ssh-keygen -Y sign -f <key> -n pose-action-answer
func cmdActionStatement(root string, args []string, stdout, stderr io.Writer) int {
	const usage = "Usage: pose action statement <act-id> --actor <principal> --answer <answer> --idempotency-key <key> [--reason <text>]"
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return usageError(stderr, usage)
	}
	id, actor, answer, key, reason := args[0], "", "", "", ""
	operation, successor, successorDigest, requestDigest := posemodel.ActionEventAnswered, "", "", ""
	role := ""
	revision := 0
	for i := 1; i < len(args); i++ {
		if i+1 >= len(args) {
			return usageError(stderr, usage)
		}
		switch args[i] {
		case "--role":
			role = args[i+1]
		case "--operation":
			operation = args[i+1]
		case "--expected-revision":
			revision, _ = strconv.Atoi(args[i+1])
		case "--request-digest":
			requestDigest = args[i+1]
		case "--successor-id":
			successor = args[i+1]
		case "--successor-digest":
			successorDigest = args[i+1]
		case "--actor":
			actor = args[i+1]
		case "--answer":
			answer = args[i+1]
		case "--idempotency-key":
			key = args[i+1]
		case "--reason":
			reason = args[i+1]
		default:
			return usageError(stderr, usage)
		}
		i++
	}
	if actor == "" || key == "" || operation == posemodel.ActionEventAnswered && answer == "" {
		return usageError(stderr, usage)
	}
	view, err := posemodel.Store{Root: root}.LoadActionRequest(id)
	if err != nil {
		render(stdout, stderr).Failure("pose action statement: " + err.Error())
		return 1
	}
	// Raw bytes, not rendered fields: what is printed is what gets signed.
	if operation == posemodel.ActionEventAnswered {
		if revision == 0 {
			revision = view.Revision
		}
		if requestDigest == "" {
			requestDigest = view.Request.RequestDigest
		}
		if _, err := (posemodel.Store{Root: root}).PreflightActionResolution(posemodel.ActionResolution{RequestID: id, Type: operation, Actor: actor, Role: role, Answer: answer, Reason: reason, IdempotencyKey: key, RequestDigest: requestDigest, ExpectedRevision: revision}, time.Now()); err != nil {
			render(stdout, stderr).Failure("pose action statement: " + err.Error())
			return 1
		}
		_, _ = stdout.Write(posemodel.AnswerStatement(view.Request, actor, answer, key, reason).Canonical())
	} else {
		if operation != posemodel.ActionEventWaived && operation != posemodel.ActionEventCancelled && operation != posemodel.ActionEventSuperseded {
			return usageError(stderr, "unsupported lifecycle operation")
		}
		if revision < 1 || reason == "" {
			return usageError(stderr, "lifecycle statement needs --expected-revision and --reason")
		}
		if requestDigest == "" {
			requestDigest = view.Request.RequestDigest
		}
		_, _ = stdout.Write(posemodel.LifecycleStatement(view.Request, posemodel.ActionResolution{Type: operation, Actor: actor, RequestDigest: requestDigest, ExpectedRevision: revision, IdempotencyKey: key, Reason: reason, SuccessorID: successor, SuccessorDigest: successorDigest}).Canonical())
	}
	return 0
}
