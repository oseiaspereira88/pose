package cli

// Docs wiki sync (spec harne8-platform-wiki-sync): `docs-sync export`
// builds the deterministic bundle (write-nothing, just serializes
// pose.Store.BuildDocsSyncBundle's output); `docs-sync push` sends it to
// the Conductor's ingestion endpoint over HTTP. Push failures are always
// visible in the refresh-log (R3) — never silent, same contract as the
// other hook-adjacent consumers in this file family.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

func cmdDocsSync(root string, args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	if len(args) == 0 {
		fmt.Fprintln(stderr, cliText(locale,
			"Usage: pose docs-sync export [--project <id>] [--out <file>]|push --url <conductor-url> --project <id> --token <token> [--from <file>]",
			"Uso: pose docs-sync export [--project <id>] [--out <arquivo>]|push --url <url-conductor> --project <id> --token <token> [--from <arquivo>]"))
		return 2
	}
	switch args[0] {
	case "export":
		return docsSyncExport(root, args[1:], stdout, stderr, locale)
	case "push":
		return docsSyncPush(root, args[1:], stdout, stderr, locale)
	default:
		fmt.Fprintf(stderr, cliText(locale, "Error: invalid argument: %s\n", "Erro: argumento inválido: %s\n"), args[0])
		return 2
	}
}

func docsSyncExport(root string, args []string, stdout, stderr io.Writer, locale cliLocale) int {
	var projectID, out string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--project":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, cliText(locale, "Error: --project requires an id", "Erro: --project exige um id"))
				return 2
			}
			i++
			projectID = args[i]
		case "--out":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, cliText(locale, "Error: --out requires a file path", "Erro: --out exige um caminho de arquivo"))
				return 2
			}
			i++
			out = args[i]
		default:
			fmt.Fprintf(stderr, cliText(locale, "Error: invalid argument: %s\n", "Erro: argumento inválido: %s\n"), args[i])
			return 2
		}
	}
	bundle, err := buildDocsSyncBundle(root, projectID)
	if err != nil {
		fmt.Fprintf(stderr, "pose docs-sync export: %v\n", err)
		return 1
	}
	encoded, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "pose docs-sync export: %v\n", err)
		return 1
	}
	if out == "" {
		fmt.Fprintln(stdout, string(encoded))
		return 0
	}
	if err := os.WriteFile(out, append(encoded, '\n'), 0o644); err != nil {
		fmt.Fprintf(stderr, "pose docs-sync export: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, cliText(locale, "Bundle written: %s (%d docs, hash %s)\n",
		"Bundle gravado: %s (%d docs, hash %s)\n"), out, len(bundle.Docs), bundle.BundleHash)
	return 0
}

func docsSyncPush(root string, args []string, stdout, stderr io.Writer, locale cliLocale) int {
	var url, projectID, token, from, trigger string
	trigger = "manual"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--url":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, cliText(locale, "Error: --url requires a value", "Erro: --url exige um valor"))
				return 2
			}
			i++
			url = args[i]
		case "--project":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, cliText(locale, "Error: --project requires an id", "Erro: --project exige um id"))
				return 2
			}
			i++
			projectID = args[i]
		case "--token":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, cliText(locale, "Error: --token requires a value", "Erro: --token exige um valor"))
				return 2
			}
			i++
			token = args[i]
		case "--from":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, cliText(locale, "Error: --from requires a file path", "Erro: --from exige um caminho de arquivo"))
				return 2
			}
			i++
			from = args[i]
		case "--ci":
			trigger = "ci"
		default:
			fmt.Fprintf(stderr, cliText(locale, "Error: invalid argument: %s\n", "Erro: argumento inválido: %s\n"), args[i])
			return 2
		}
	}
	if url == "" || projectID == "" {
		fmt.Fprintln(stderr, cliText(locale, "Error: --url and --project are required", "Erro: --url e --project são obrigatórios"))
		return 2
	}

	var payload []byte
	if from != "" {
		raw, err := os.ReadFile(from)
		if err != nil {
			fmt.Fprintf(stderr, "pose docs-sync push: %v\n", err)
			return 1
		}
		payload = raw
	} else {
		bundle, err := buildDocsSyncBundle(root, projectID)
		if err != nil {
			logDocsSyncPushOutcome(root, trigger, projectID, err)
			fmt.Fprintf(stderr, "pose docs-sync push: %v\n", err)
			return 1
		}
		encoded, err := json.Marshal(bundle)
		if err != nil {
			logDocsSyncPushOutcome(root, trigger, projectID, err)
			fmt.Fprintf(stderr, "pose docs-sync push: %v\n", err)
			return 1
		}
		payload = encoded
	}

	endpoint := url + "/api/v1/projects/" + projectID + "/wiki/bundle"
	req, err := http.NewRequest(http.MethodPut, endpoint, bytes.NewReader(payload))
	if err != nil {
		logDocsSyncPushOutcome(root, trigger, projectID, err)
		fmt.Fprintf(stderr, "pose docs-sync push: %v\n", err)
		return 1
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logDocsSyncPushOutcome(root, trigger, projectID, err)
		fmt.Fprintf(stderr, "pose docs-sync push: %v\n", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("conductor returned %d: %s", resp.StatusCode, string(body))
		logDocsSyncPushOutcome(root, trigger, projectID, err)
		fmt.Fprintf(stderr, "pose docs-sync push: %v\n", err)
		return 1
	}
	logDocsSyncPushOutcome(root, trigger, projectID, nil)
	fmt.Fprintf(stdout, cliText(locale, "Bundle pushed to %s\n", "Bundle enviado para %s\n"), endpoint)
	return 0
}

func buildDocsSyncBundle(root, projectID string) (*pose.DocsSyncBundle, error) {
	store := pose.Store{Root: root}
	return store.BuildDocsSyncBundle(context.Background(), projectID)
}

// logDocsSyncPushOutcome makes push failures visible in the shared
// refresh-log (R3) — the same "never silent" contract the other hook-
// adjacent consumers in this package already honor.
func logDocsSyncPushOutcome(root, trigger, projectID string, pushErr error) {
	entry := refreshLogEntry{
		At: time.Now().UTC().Format(time.RFC3339), Trigger: trigger, Target: projectID,
		Consumer: "docs-sync", Result: "ok",
	}
	if pushErr != nil {
		entry.Result = "failed"
		entry.Error = pushErr.Error()
	}
	_ = appendRefreshLog(root, entry)
}
