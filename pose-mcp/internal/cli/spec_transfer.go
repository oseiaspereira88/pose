package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

const specTransferUsage = "Usage: pose spec-transfer <preview|apply|resume|status> [options]"

func cmdSpecTransfer(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return transferUsage(stderr, "a subcommand is required")
	}
	command, flags := args[0], args[1:]
	values := map[string][]string{}
	for i := 0; i < len(flags); i++ {
		flag := flags[i]
		if flag == "--json" {
			values[flag] = append(values[flag], "true")
			continue
		}
		if !strings.HasPrefix(flag, "--") || i+1 >= len(flags) || strings.HasPrefix(flags[i+1], "--") {
			return transferUsage(stderr, "invalid option or missing value: "+flag)
		}
		switch flag {
		case "--source", "--destination", "--map", "--date", "--plan", "--digest", "--authorize-project", "--operation", "--project":
			values[flag] = append(values[flag], flags[i+1])
			i++
		default:
			return transferUsage(stderr, "unknown option: "+flag)
		}
	}
	resolver, currentProject, err := posemodel.EnvironmentArtifactResolver(root, "")
	if err != nil {
		return transferFailure(stderr, "pose spec-transfer: "+err.Error())
	}
	switch command {
	case "preview":
		source, err := requiredTransferRef(values, "--source")
		if err != nil {
			return transferUsage(stderr, err.Error())
		}
		destination, err := requiredTransferRef(values, "--destination")
		if err != nil {
			return transferUsage(stderr, err.Error())
		}
		mappings, err := parseTransferMappings(values["--map"])
		if err != nil {
			return transferUsage(stderr, err.Error())
		}
		date, err := oneTransferValue(values, "--date", false)
		if err != nil {
			return transferUsage(stderr, err.Error())
		}
		plan, err := posemodel.PreviewSpecTransfer(resolver, posemodel.SpecTransferRequest{Source: source, Destination: destination, Mappings: mappings}, date)
		if err != nil {
			return transferFailure(stderr, "pose spec-transfer preview: "+err.Error())
		}
		return writeTransferJSON(stdout, stderr, plan)
	case "apply":
		planPath, err := oneTransferValue(values, "--plan", true)
		if err != nil {
			return transferUsage(stderr, err.Error())
		}
		digest, err := oneTransferValue(values, "--digest", true)
		if err != nil {
			return transferUsage(stderr, err.Error())
		}
		rawPlan, err := readTransferPlanFile(planPath)
		if err != nil {
			return transferFailure(stderr, "pose spec-transfer apply: "+err.Error())
		}
		plan, err := decodeTransferPlan(rawPlan)
		if err != nil {
			return transferFailure(stderr, "pose spec-transfer apply: invalid plan")
		}
		status, err := posemodel.ApplySpecTransfer(resolver, plan, digest, authorizedTransferProjects(values), nil)
		if err != nil {
			return transferFailure(stderr, "pose spec-transfer apply: "+err.Error())
		}
		return writeTransferJSON(stdout, stderr, status)
	case "resume":
		operation, err := oneTransferValue(values, "--operation", true)
		if err != nil {
			return transferUsage(stderr, err.Error())
		}
		project, err := oneTransferValue(values, "--project", true)
		if err != nil {
			return transferUsage(stderr, err.Error())
		}
		status, err := posemodel.ResumeSpecTransfer(resolver, project, operation, authorizedTransferProjects(values), nil)
		if err != nil {
			return transferFailure(stderr, "pose spec-transfer resume: "+err.Error())
		}
		return writeTransferJSON(stdout, stderr, status)
	case "status":
		operation, err := oneTransferValue(values, "--operation", true)
		if err != nil {
			return transferUsage(stderr, err.Error())
		}
		project, err := oneTransferValue(values, "--project", false)
		if err != nil {
			return transferUsage(stderr, err.Error())
		}
		if project == "" {
			project = currentProject
		}
		store, err := resolver.Roots.StoreFor(project)
		if err != nil {
			return transferFailure(stderr, "pose spec-transfer status: "+err.Error())
		}
		status, err := posemodel.ReadSpecTransferStatus(store, operation, project)
		if err != nil {
			return transferFailure(stderr, "pose spec-transfer status: "+err.Error())
		}
		return writeTransferJSON(stdout, stderr, status)
	default:
		return transferUsage(stderr, "unknown subcommand: "+command)
	}
}

func requiredTransferRef(values map[string][]string, name string) (posemodel.ArtifactRef, error) {
	value, err := oneTransferValue(values, name, true)
	if err != nil {
		return posemodel.ArtifactRef{}, err
	}
	ref, err := posemodel.ParseArtifactRef(value)
	if err != nil || ref.Project == "" || ref.Kind != "spec" {
		return posemodel.ArtifactRef{}, fmt.Errorf("%s requires a qualified xref project spec identity", name)
	}
	return ref, nil
}

func parseTransferMappings(values []string) ([]posemodel.SpecTransferRequirementMapping, error) {
	out := make([]posemodel.SpecTransferRequirementMapping, 0, len(values))
	for _, value := range values {
		left, right, ok := strings.Cut(value, "=")
		if !ok || left == "" || right == "" {
			return nil, fmt.Errorf("--map must use source=disposition[:destination] (use - for an empty requirement)")
		}
		disposition, destination, hasDestination := strings.Cut(right, ":")
		if !hasDestination {
			destination = "-"
		}
		if left == "-" {
			left = ""
		}
		if destination == "-" {
			destination = ""
		}
		out = append(out, posemodel.SpecTransferRequirementMapping{SourceRequirement: left, DestinationRequirement: destination, Disposition: disposition})
	}
	return out, nil
}

func oneTransferValue(values map[string][]string, key string, required bool) (string, error) {
	items := values[key]
	if len(items) > 1 || required && (len(items) != 1 || strings.TrimSpace(items[0]) == "") {
		return "", fmt.Errorf("%s must be supplied exactly once", key)
	}
	if len(items) == 0 {
		return "", nil
	}
	return items[0], nil
}

func authorizedTransferProjects(values map[string][]string) map[string]bool {
	projects := map[string]bool{}
	for _, id := range values["--authorize-project"] {
		projects[id] = true
	}
	return projects
}

func readTransferPlanFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > 2<<20 {
		return nil, fmt.Errorf("plan file unavailable or exceeds 2 MiB")
	}
	return os.ReadFile(path)
}

func decodeTransferPlan(raw []byte) (posemodel.SpecTransferPlan, error) {
	var plan posemodel.SpecTransferPlan
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&plan); err != nil {
		return plan, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return plan, fmt.Errorf("trailing JSON")
		}
		return plan, err
	}
	return plan, nil
}

func writeTransferJSON(w, stderr io.Writer, value any) int {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		transferFailure(stderr, "pose spec-transfer: "+err.Error())
		return 1
	}
	return 0
}

func transferUsage(stderr io.Writer, reason string) int {
	r := render(io.Discard, stderr)
	r.Usage(specTransferUsage)
	r.Failure(reason)
	return 2
}

func transferFailure(stderr io.Writer, message string) int {
	render(io.Discard, stderr).Failure(message)
	return 1
}
