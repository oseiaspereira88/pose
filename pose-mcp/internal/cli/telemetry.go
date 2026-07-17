package cli

// Opt-in anonymous telemetry (spec pose-telemetry-optin).
//
// Privacy contract:
//   - OFF by default; only `pose telemetry enable` turns it on.
//   - Even when enabled, nothing is sent unless POSE_TELEMETRY_URL is set
//     (no default collection endpoint is baked into the binary).
//   - Payload is only: anonymous instance id, binary version, subcommand
//     name. Never repo names, paths, spec content or user data.
//   - Emission is best-effort with a hard timeout; failures are silent and
//     never affect the command's exit code.

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type telemetryConfig struct {
	Enabled bool   `json:"enabled"`
	AnonID  string `json:"anon_id"`
}

func telemetryPath(root string) string {
	return filepath.Join(root, ".pose", "telemetry.json")
}

func loadTelemetry(root string) telemetryConfig {
	var cfg telemetryConfig
	b, err := os.ReadFile(telemetryPath(root))
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(b, &cfg)
	return cfg
}

func cmdTelemetry(args []string, stdout, stderr io.Writer) int {
	root, err := projectRoot()
	if err != nil {
		fmt.Fprintf(stderr, "pose telemetry: %v\n", err)
		return 1
	}
	sub := "status"
	if len(args) > 0 {
		sub = args[0]
	}
	cfg := loadTelemetry(root)
	switch sub {
	case "status":
		state := "disabled"
		if cfg.Enabled {
			state = "enabled"
		}
		endpoint := os.Getenv("POSE_TELEMETRY_URL")
		if endpoint == "" {
			endpoint = "(unset — nothing is ever sent)"
		}
		fmt.Fprintf(stdout, "telemetry: %s\nendpoint:  %s\n", state, endpoint)
		if cfg.Enabled {
			fmt.Fprintf(stdout, "anon_id:   %s\n", cfg.AnonID)
		}
		fmt.Fprintln(stdout, "payload:   anon_id, version, subcommand — never paths, repos or content")
		return 0
	case "enable":
		if cfg.AnonID == "" {
			b := make([]byte, 8)
			if _, err := rand.Read(b); err != nil {
				fmt.Fprintf(stderr, "pose telemetry: %v\n", err)
				return 1
			}
			cfg.AnonID = hex.EncodeToString(b)
		}
		cfg.Enabled = true
	case "disable":
		cfg.Enabled = false
	default:
		fmt.Fprintf(stderr, "pose telemetry: unknown subcommand %q (use enable|disable|status)\n", sub)
		return 2
	}
	if err := os.MkdirAll(filepath.Join(root, ".pose"), 0o755); err != nil {
		fmt.Fprintf(stderr, "pose telemetry: %v\n", err)
		return 1
	}
	b, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(telemetryPath(root), append(b, '\n'), 0o644); err != nil {
		fmt.Fprintf(stderr, "pose telemetry: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "telemetry %sd\n", sub)
	return 0
}

// emitTelemetry sends a fire-and-forget usage event. It is a no-op unless the
// instance opted in AND POSE_TELEMETRY_URL is set.
func emitTelemetry(command string) {
	url := os.Getenv("POSE_TELEMETRY_URL")
	if url == "" {
		return
	}
	root, err := projectRoot()
	if err != nil {
		return
	}
	cfg := loadTelemetry(root)
	if !cfg.Enabled {
		return
	}
	payload, _ := json.Marshal(map[string]string{
		"anon_id": cfg.AnonID,
		"version": Version,
		"command": command,
	})
	client := &http.Client{Timeout: 800 * time.Millisecond}
	resp, err := client.Post(url, "application/json", bytes.NewReader(payload))
	if err == nil {
		_ = resp.Body.Close()
	}
}
