package pose

import "fmt"

// Never interpolate an arbitrary capture error: session persistence errors may
// carry private paths, and future adapters may include raw output in errors.
func delegationCaptureFailure(agent, events, stage string, err error) string {
	reason := "capture failed (private error details withheld)"
	switch err.Error() {
	case "ambiguous adapter capture", "adapter capture is not a bounded reference",
		"conflicting adapter capture channels", "adapter changed the captured session",
		"adapter reset instant is not RFC3339", "redacted capture exceeds byte budget":
		reason = err.Error()
	}
	message := fmt.Sprintf("the agent's output could not be redacted; it was not kept (agent=%q events=%q stage=%q: %s)", agent, events, stage, reason)
	if reason == "ambiguous adapter capture" {
		message += "; the session/reset pattern must capture one reference per channel"
		if events == "claude-stream-json" {
			message += "; for a session_id repeated across events, restrict the session pattern to the system/init event"
		}
	}
	return message
}
