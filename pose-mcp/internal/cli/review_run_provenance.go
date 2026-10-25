package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func cmdReviewRunSign(root string, args []string, stdout, stderr io.Writer) int {
	usage := "Usage: pose review run-sign <run-id> --issuer <name> [--ttl <duration>] [--apply] [--json]"
	id, issuer := "", ""
	ttl := time.Duration(0)
	apply, jsonOut := false, false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--issuer", "--ttl":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return usageError(stderr, usage)
			}
			flag := args[i]
			i++
			if flag == "--issuer" {
				issuer = args[i]
			} else {
				var e error
				ttl, e = time.ParseDuration(args[i])
				if e != nil || ttl <= 0 {
					return usageError(stderr, usage)
				}
			}
		case "--apply":
			apply = true
		case "--json":
			jsonOut = true
		default:
			if id != "" || strings.HasPrefix(args[i], "-") {
				return usageError(stderr, usage)
			}
			id = args[i]
		}
	}
	if id == "" || issuer == "" {
		return usageError(stderr, usage)
	}
	key, e := posemodel.LoadIssuerKey(root, issuer)
	if e != nil {
		render(stdout, stderr).Failure("pose review run proof: " + strings.TrimPrefix(e.Error(), "pose: "))
		return 1
	}
	proof, e := cliGovernedStore(root).SignReviewRunProof(id, key, ttl, time.Now(), apply)
	if e != nil {
		render(stdout, stderr).Failure("pose review run proof: " + strings.TrimPrefix(e.Error(), "pose: "))
		return 1
	}
	if jsonOut {
		if e = json.NewEncoder(stdout).Encode(proof); e != nil {
			return 1
		}
	} else {
		out := render(stdout, stderr)
		out.Field("review_run_proof.run", id)
		out.Field("review_run_proof.id", proof.ProofID)
		out.Field("review_run_proof.apply", fmt.Sprint(apply))
		out.Field("review_run_proof.assurance", "native signed executor declaration; not a vendor receipt or human confirmation")
	}
	return 0
}
func cmdReviewRunProof(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 || len(args) > 3 || len(args) == 3 && args[2] != "--json" {
		return usageError(stderr, "Usage: pose review run-proof <run-id> <proof-id> [--json]")
	}
	proof, e := cliGovernedStore(root).VerifyReviewRunProof(args[0], args[1])
	if e != nil {
		render(stdout, stderr).Failure("pose review run proof: " + strings.TrimPrefix(e.Error(), "pose: "))
		return 1
	}
	if len(args) == 3 {
		if e = json.NewEncoder(stdout).Encode(proof); e != nil {
			return 1
		}
	} else {
		out := render(stdout, stderr)
		out.Field("review_run_proof.run", proof.RunID)
		out.Field("review_run_proof.assurance", proof.Assurance)
		out.Field("review_run_proof.state", proof.State)
		out.Field("review_run_proof.draft_decision", proof.DraftDecision)
		out.Field("review_run_proof.complete_review", fmt.Sprint(proof.CompleteReview))
		out.Field("review_run_proof.output_complete", fmt.Sprint(proof.OutputComplete))
		out.Field("review_run_proof.source", proof.IdentitySource)
		out.Field("review_run_proof.reviewer", proof.Reviewer.Vendor+"/"+proof.Reviewer.Model)
		out.Field("review_run_proof.implementation", proof.Implementation.Vendor+"/"+proof.Implementation.Model)
	}
	return 0
}
