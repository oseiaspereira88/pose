package cli

// pose update --dry-run (spec pose-update-dry-run-reports-the-whole-update).
//
// The dry-run used to print only schema migrations and "no changes applied",
// while a real update merged manuals and machinery, stamped engine_version,
// seeded project identity and opened a configuration review. Four instances
// were declared current on that output while still stamped by an older
// engine.
//
// It then ran the update on a disposable copy of the instance. Ten review
// rounds each found a layout the copy reproduced differently — links, modes,
// unreadable files, a directory that can be searched but not listed — and the
// copy never held the source tree module discovery reads. The preview now runs
// the update's own steps through an operation-scoped view (decision D2, ADR
// 2026-10-22): reads see the instance as it is, read lazily and by name, with
// every change the update stages on top; no change leaves memory. What the
// preview reports is what those steps did to the view, in the update's own
// words.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/version"
)

// previewUpdate runs the update's steps against a view of the instance and
// reports what they would change, without changing anything.
func previewUpdate(u instanceUpdate, now time.Time, stdout, stderr io.Writer) int {
	text := u.text
	r := render(stdout, stderr)
	// The running engine is the one previewed: a preview never looks for a
	// newer release, downloads or runs it (spec decision D2).
	r.ContractLine("[DRY-RUN] " + fmt.Sprintf(text(
		"previewing this engine's update of the instance (POSE %s), as `pose update --no-self` runs it; without --no-self, `pose update` first looks for a newer release and, when there is one, that engine runs the update instead — this preview neither downloads nor runs it",
		"prévia da atualização da instância por este motor (POSE %s), como `pose update --no-self` a executa; sem --no-self, `pose update` primeiro procura uma versão mais nova e, havendo uma, é esse motor que executa a atualização — esta prévia não a baixa nem a executa"),
		version.ReleaseBase()))

	preview := posemodel.NewPreviewFiles(now)
	engineBefore := recordedEngineVersionIn(preview, u.root)
	lines := &previewLines{out: stdout}
	code := u.apply(preview, lines, lines)
	lines.Close()

	changes := preview.Changes(u.root)
	requests := 0
	for _, change := range changes {
		if change.Verb == "create" && isActionRequestFile(change.Path) {
			requests++
		}
		r.ContractLine("[DRY-RUN] " + previewChangeLine(change, text))
	}
	if requests > 0 {
		r.ContractLine("[DRY-RUN] " + text(
			"the request ids above are what opening them now would produce; the update assigns its own ids when it opens them",
			"os ids de pedido acima são os que abri-los agora produziria; a atualização atribui os seus ao abri-los"))
	}
	if engineAfter := recordedEngineVersionIn(preview, u.root); engineAfter != engineBefore {
		r.ContractLine(fmt.Sprintf("[DRY-RUN] %s %s -> %s", text("would stamp engine_version:", "carimbaria engine_version:"), orDash(engineBefore), engineAfter))
	}
	for _, op := range preview.Operations() {
		if op.Refused != "" {
			r.ContractLine("[DRY-RUN] " + text("refused now, as the system reports it: ", "recusado agora, como o sistema informa: ") + op.Refused)
		}
	}
	gaps := preview.Gaps()
	for _, gap := range gaps {
		r.ContractLine("[DRY-RUN] " + text("not evaluated: ", "não avaliado: ") + gap)
	}
	r.ContractLine("[DRY-RUN] " + text(
		"the preview runs the update's own steps against what it observed now; the update itself can still meet a concurrent change, a full disk, a refusal the system does not report in advance or a failing provider",
		"a prévia executa os passos da própria atualização sobre o que observou agora; a atualização ainda pode encontrar uma mudança concorrente, um disco cheio, uma recusa que o sistema não informa antes ou um provedor falhando"))

	switch {
	case len(gaps) > 0:
		// An incomplete account is never reported as the whole update, nor as
		// one that changes nothing.
		r.ContractLine(fmt.Sprintf(text(
			"Result: DRY-RUN — INCOMPLETE: %d step(s) could not be evaluated (above), so this is not the whole update; nothing was applied.",
			"Resultado: DRY-RUN — INCOMPLETO: %d passo(s) não puderam ser avaliados (acima), então isto não é a atualização inteira; nada foi aplicado."), len(gaps)))
		if code == 0 {
			return 1
		}
		return code
	case code != 0:
		r.ContractLine(text("Result: DRY-RUN — the update would fail; nothing was applied.", "Resultado: DRY-RUN — a atualização falharia; nada foi aplicado."))
		return code
	case len(changes) == 0:
		r.ContractLine(text("Result: DRY-RUN — the update would change nothing.", "Resultado: DRY-RUN — a atualização não mudaria nada."))
		return 0
	}
	r.ContractLine(fmt.Sprintf(text("Result: DRY-RUN — the update would change %d path(s); nothing was applied.", "Resultado: DRY-RUN — a atualização mudaria %d caminho(s); nada foi aplicado."), len(changes)))
	return 0
}

var previewVerbsPtBR = map[string]string{
	"create": "criaria", "modify": "modificaria", "remove": "removeria", "change mode": "mudaria o modo de",
	"create directory": "criaria o diretório", "remove directory": "removeria o diretório",
	"create link": "criaria o link", "replace link": "substituiria o link",
	"replace link with file": "substituiria o link por arquivo", "replace file with link": "substituiria o arquivo por link",
	"replace link with directory": "substituiria o link por diretório", "replace file with directory": "substituiria o arquivo por diretório",
	"replace directory with file": "substituiria o diretório por arquivo", "replace directory with link": "substituiria o diretório por link",
}

// previewChangeLine names one net change. A change made through a link names
// the link; one whose location is outside the instance says so.
func previewChangeLine(change posemodel.PreviewChange, text func(string, string) string) string {
	verb := "would " + change.Verb
	if pt, ok := previewVerbsPtBR[change.Verb]; ok {
		verb = text(verb, pt)
	}
	if change.Outside {
		verb += text(" outside the instance", " fora da instância")
	}
	line := verb + ": " + change.Path
	if change.Through != "" {
		line += text(" (through ", " (por meio de ") + change.Through + ")"
	}
	return line
}

// previewLines prefixes each line the previewed update prints, from either
// stream, in the order printed. The update's own "Result:" line is left out:
// the preview ends with its own.
type previewLines struct {
	mu      sync.Mutex
	out     io.Writer
	pending []byte
}

func (p *previewLines) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending = append(p.pending, b...)
	for {
		i := bytes.IndexByte(p.pending, '\n')
		if i < 0 {
			return len(b), nil
		}
		p.emit(string(p.pending[:i]))
		p.pending = p.pending[i+1:]
	}
}

// Close prints a last line the update left without a newline.
func (p *previewLines) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.pending) > 0 {
		p.emit(string(p.pending))
		p.pending = nil
	}
}

func (p *previewLines) emit(line string) {
	line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
	if line == "" || strings.HasPrefix(line, "Result:") || strings.HasPrefix(line, "Resultado:") {
		return
	}
	fmt.Fprintln(p.out, "[DRY-RUN] "+line)
}

func isActionRequestFile(path string) bool {
	return strings.HasPrefix(path, ".pose/actions/act-") && strings.HasSuffix(path, ".jsonl") && !strings.Contains(strings.TrimPrefix(path, ".pose/actions/"), "/")
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// recordedEngineVersion is the engine that last updated the instance.
func recordedEngineVersion(root string) string {
	return recordedEngineVersionIn(posemodel.NativeFiles(), root)
}

// recordedEngineVersionIn reads it through files: a preview reads the stamp
// the update staged.
func recordedEngineVersionIn(files posemodel.Files, root string) string {
	raw, err := files.ReadFile(filepath.Join(root, ".pose", "state", "machinery-manifest.json"))
	if err != nil {
		return ""
	}
	var doc struct {
		EngineVersion string `json:"engine_version"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return ""
	}
	return doc.EngineVersion
}

// staleInstanceEngine returns the engine that last updated the instance when
// it is older than the running engine, or "". Pre-release suffixes are
// ignored: a 7.3.0-dev build is the 7.3.0 engine.
func staleInstanceEngine(root string) string {
	return staleInstanceEngineIn(posemodel.NativeFiles(), root)
}

// staleInstanceEngineIn is staleInstanceEngine through files, so a gate a
// preview runs judges the stamp the update would leave.
func staleInstanceEngineIn(files posemodel.Files, root string) string {
	recorded := recordedEngineVersionIn(files, root)
	if recorded == "" {
		return ""
	}
	parse := func(v string) [3]int {
		var out [3]int
		v = strings.TrimPrefix(strings.SplitN(v, "-", 2)[0], "v")
		for i, part := range strings.SplitN(v, ".", 3) {
			fmt.Sscanf(part, "%d", &out[i])
		}
		return out
	}
	have, want := parse(recorded), parse(version.ReleaseBase())
	for i := 0; i < 3; i++ {
		if have[i] != want[i] {
			if have[i] < want[i] {
				return recorded
			}
			return ""
		}
	}
	return ""
}
