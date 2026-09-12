package cliout

// The message catalog (spec pose-cli-output-rendering-system R10).
//
// Every line this package emits is keyed by an id, the way help_catalog.go
// already keys help. Two things follow: bilingual parity is a test rather than a
// promise, and a machine has a stable key that survives rewording — the human
// text may change in a release, the id may not.

import "fmt"

const (
	LocaleEN   = "en"
	LocalePtBR = "pt-BR"
)

type MessageID string

const (
	MsgResultLabel MessageID = "result.label"
	MsgResultOK    MessageID = "result.ok"
	MsgResultFail  MessageID = "result.fail"

	MsgStatePass    MessageID = "state.pass"
	MsgStateFail    MessageID = "state.fail"
	MsgStateError   MessageID = "state.error"
	MsgStateWarning MessageID = "state.warning"
	MsgStateSkipped MessageID = "state.skipped"
	MsgStateInfo    MessageID = "state.info"
	MsgStateHint    MessageID = "state.hint"

	MsgFix       MessageID = "finding.fix"
	MsgUsage     MessageID = "error.usage"
	MsgError     MessageID = "error.label"
	MsgSuggest   MessageID = "error.suggest"
	MsgUnknown   MessageID = "error.unknown"
	MsgStepsDone MessageID = "steps.summary"
	MsgFullOut   MessageID = "steps.full_output"
	MsgElapsed   MessageID = "steps.elapsed"
)

type message struct {
	en   string
	ptBR string
}

// catalog holds one entry per id. A missing translation is a test failure, not
// a silent fallback to English.
var catalog = map[MessageID]message{
	MsgResultLabel: {"Result", "Resultado"},
	MsgResultOK:    {"SUCCESS", "SUCESSO"},
	MsgResultFail:  {"FAILURE", "FALHA"},

	MsgStatePass:    {"pass", "passou"},
	MsgStateFail:    {"fail", "falhou"},
	MsgStateError:   {"error", "erro"},
	MsgStateWarning: {"warning", "aviso"},
	MsgStateSkipped: {"skipped", "ignorado"},
	MsgStateInfo:    {"info", "info"},
	MsgStateHint:    {"hint", "dica"},

	MsgFix:     {"fix", "correção"},
	MsgUsage:   {"Usage", "Uso"},
	MsgError:   {"Error", "Erro"},
	MsgSuggest: {"did you mean %q?", "você quis dizer %q?"},
	MsgUnknown: {"unknown %s: %q", "%s desconhecido: %q"},

	MsgStepsDone: {"%d step(s) · %s · %s", "%d passo(s) · %s · %s"},
	MsgFullOut:   {"full output in %s", "saída completa em %s"},
	MsgElapsed:   {"%.1fs", "%.1fs"},
}

// Msg renders one catalog entry in the reader's language.
func Msg(id MessageID, locale string, args ...any) string {
	entry, ok := catalog[id]
	if !ok {
		// An unknown id is a bug in the caller, and saying so beats printing
		// nothing where a reader expects a sentence.
		return string(id)
	}
	text := entry.en
	if locale == LocalePtBR && entry.ptBR != "" {
		text = entry.ptBR
	}
	if len(args) == 0 {
		return text
	}
	return fmt.Sprintf(text, args...)
}

// CatalogIDs lists every id, for the parity test and for tooling.
func CatalogIDs() []MessageID {
	ids := make([]MessageID, 0, len(catalog))
	for id := range catalog {
		ids = append(ids, id)
	}
	return ids
}

// CatalogEntry exposes both translations of one id.
func CatalogEntry(id MessageID) (en, ptBR string, ok bool) {
	entry, found := catalog[id]
	return entry.en, entry.ptBR, found
}
