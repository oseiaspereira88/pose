package cli

import "encoding/json"

// validationCheck is the shell-free portion of a matrix entry. Legacy
// `command` entries deliberately remain outside this type until parity is
// established; native execution consumes only Program, Args and Env.
type validationCheck struct {
	Name     string            `json:"name"`
	Program  string            `json:"program"`
	Args     []string          `json:"args"`
	Env      map[string]string `json:"env"`
	Severity string            `json:"severity"`
}

// parseStructuredChecks rejects malformed JSON before any command can run.
func parseStructuredChecks(raw []byte) ([]validationCheck, error) {
	var doc struct {
		Stacks map[string]struct {
			Checks []validationCheck `json:"checks"`
		} `json:"stacks"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	var checks []validationCheck
	for _, stack := range doc.Stacks {
		for _, check := range stack.Checks {
			if check.Program != "" {
				checks = append(checks, check)
			}
		}
	}
	return checks, nil
}
