package cli

import "sort"

// FlagHelp describes a single command flag or option.
type FlagHelp struct {
	Flag            string
	DescriptionEN   string
	DescriptionPtBR string
}

// SubcommandHelp describes a subcommand within a command group.
type SubcommandHelp struct {
	Name        string
	Usage       string
	SummaryEN   string
	SummaryPtBR string
}

// CommandHelp contains full structured documentation for a CLI command.
type CommandHelp struct {
	Name            string
	SummaryEN       string
	SummaryPtBR     string
	Usage           string
	DescriptionEN   string
	DescriptionPtBR string
	Flags           []FlagHelp
	Subcommands     []SubcommandHelp
	Examples        []string
}

// knownCommandNames lists the commands the help catalog documents, sorted, so an
// unknown command can be matched against real names instead of guessed at
// (spec pose-cli-output-rendering-system R11).
func knownCommandNames() []string {
	names := make([]string, 0, len(commandHelpCatalog))
	for name := range commandHelpCatalog {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// commandHelpCatalog maps command names to their structured help definitions.
var commandHelpCatalog = map[string]CommandHelp{
	"init": {
		Name:            "init",
		SummaryEN:       "Install POSE in the current repository, or confirm it is installed",
		SummaryPtBR:     "Instala o POSE no repositório atual, ou confirma que já está instalado",
		Usage:           "pose init [--wizard [--yes]] [--locale <tag>] [--project-name <name>] [--project-id <id>] [--skip-mcp] [--allow-non-git]",
		DescriptionEN:   "Where no POSE instance exists, runs the full installation into the current repository — the same instance `pose install .` produces, ready for `pose check --strict` and `pose new-spec`. On an installed instance it only ensures the directory structure, writes nothing else and names `pose update` to refresh it. The installer's flags pass through. --wizard then detects stack modules and seeds the validation matrix.",
		DescriptionPtBR: "Onde não há instância POSE, roda a instalação completa no repositório atual — a mesma instância que `pose install .` produz, pronta para `pose check --strict` e `pose new-spec`. Numa instância instalada, só garante a estrutura de diretórios, não grava mais nada e aponta `pose update` para atualizá-la. As flags do instalador são repassadas. --wizard depois detecta os módulos de stack e semeia a matriz de validação.",
		Flags: []FlagHelp{
			{"--wizard", "Run interactive onboarding wizard to detect modules and stack rules", "Executa o assistente de onboarding para detectar módulos e regras de stack"},
			{"--yes", "Auto-accept wizard prompts with recommended defaults", "Aceita automaticamente as perguntas do assistente com os padrões recomendados"},
			{"--locale <tag>", "Locale of the installed manuals (installer flag)", "Idioma dos manuais instalados (flag do instalador)"},
			{"--project-name <name>", "Project name to stamp (installer flag)", "Nome do projeto a carimbar (flag do instalador)"},
			{"--project-id <id>", "Project id to declare in .pose/project.json (installer flag)", "Id do projeto a declarar em .pose/project.json (flag do instalador)"},
			{"--skip-mcp", "Do not write .mcp.json (installer flag)", "Não grava o .mcp.json (flag do instalador)"},
			{"--allow-non-git", "Install outside a git repository (installer flag)", "Instala fora de um repositório git (flag do instalador)"},
		},
		Examples: []string{
			"pose init",
			"pose init --wizard --yes",
		},
	},
	"version": {
		Name:            "version",
		SummaryEN:       "Display POSE binary version and schema compatibility",
		SummaryPtBR:     "Exibe a versão do binário POSE e compatibilidade de schema",
		Usage:           "pose version [target-dir]",
		DescriptionEN:   "Prints the compiled Go binary version, Git commit SHA, and compares the instance schema version with the engine schema.",
		DescriptionPtBR: "Imprime a versão compilada do binário Go, commit SHA do Git e compara o schema da instância com o do motor.",
		Examples: []string{
			"pose version",
			"pose version /path/to/project",
		},
	},
	"context": {
		Name:            "context",
		SummaryEN:       "Resolve the selected project and qualified task context",
		SummaryPtBR:     "Resolve o projeto selecionado e o contexto da tarefa qualificada",
		Usage:           "pose context [--project-id <id>] [--task <artifact-ref>] [--json]",
		DescriptionEN:   "Reads the selected logical project, canonical task authority, revision, redirect state, and supported contracts without exposing filesystem roots or mutating project state.",
		DescriptionPtBR: "Lê o projeto lógico selecionado, a autoridade canônica da tarefa, revisão, redirecionamento e contratos suportados sem expor caminhos nem alterar o estado do projeto.",
		Flags: []FlagHelp{
			{"--project-id <id>", "Select an explicitly configured logical project", "Seleciona um projeto lógico configurado explicitamente"},
			{"--task <artifact-ref>", "Resolve a local or qualified task reference", "Resolve uma referência local ou qualificada da tarefa"},
			{"--json", "Output the path-free context as JSON", "Emite o contexto sem caminhos em JSON"},
		},
		Examples: []string{
			"pose context --json",
			"pose context --task xref:proj.executor/spec:checkout --json",
		},
	},
	"doctor": {
		Name:            "doctor",
		SummaryEN:       "Diagnose POSE installation and repository health",
		SummaryPtBR:     "Diagnostica a saúde da instalação e do repositório POSE",
		Usage:           "pose doctor [--json] [--fix [--yes]] [--only <check>]",
		DescriptionEN:   "Runs deterministic health diagnostics on dependencies (git, go), repository structure, schema version, skill symlinks, MCP configuration, and retired machinery.",
		DescriptionPtBR: "Executa diagnósticos determinísticos de saúde sobre dependências (git, go), estrutura do repositório, versão de schema, symlinks de skills, MCP e maquinário descontinuado.",
		Flags: []FlagHelp{
			{"--json", "Output diagnostics report in structured JSON format", "Emite o relatório de diagnósticos em formato JSON estruturado"},
			{"--fix", "Preview or apply automated remediation for fixable diagnostics", "Visualiza ou aplica correções automáticas para diagnósticos corrigíveis"},
			{"--yes", "Apply fix remediations without interactive confirmation", "Aplica as correções automáticas sem confirmação interativa"},
			{"--only <check>", "Run only the specified diagnostic check name", "Executa apenas o check de diagnóstico especificado"},
		},
		Examples: []string{
			"pose doctor",
			"pose doctor --fix --yes",
			"pose doctor --json",
		},
	},
	"validate": {
		Name:            "validate",
		SummaryEN:       "Execute the deterministic validation matrix across all modules",
		SummaryPtBR:     "Executa a matriz determinística de validação em todos os módulos",
		Usage:           "pose validate [--strict|--tolerant] [--stack <s>] [--module <p>] [--report] [--json-out <path>] [--verbose]",
		DescriptionEN:   "Executes deterministic verification commands (tests, linters, typechecks, builds) declared in .pose/indexes/validation-matrix.json.",
		DescriptionPtBR: "Executa comandos de verificação determinísticos (testes, linters, checagens de tipo, builds) declarados em .pose/indexes/validation-matrix.json.",
		Flags: []FlagHelp{
			{"--strict", "Fail on any error or structural validation warning (default in CI)", "Falha em qualquer erro ou aviso estrutural de validação (padrão em CI)"},
			{"--tolerant", "Allow non-blocking warnings while failing on required errors", "Permite avisos não-bloqueantes, falhando apenas em erros obrigatórios"},
			{"--stack <name>", "Filter execution to modules matching the given stack (e.g. go, node, python)", "Filtra a execução para módulos da stack informada"},
			{"--module <path>", "Filter execution to a specific module directory path", "Filtra a execução para o caminho de um módulo específico"},
			{"--report", "Persist validation findings into .pose/reports/", "Persiste os achados de validação sob .pose/reports/"},
			{"--json-out <path>", "Write structured validation outcome to the specified JSON path", "Grava o resultado da validação no caminho JSON especificado"},
			{"--json <path>", "Deprecated alias of --json-out <path>", "Alias depreciado de --json-out <path>"},
			{"--changed-from <rev>", "Validate only modules affected between git revisions", "Valida apenas módulos afetados entre as revisões git"},
			{"--verbose", "Stream each check's output as it runs; by default it is captured and a failing check's tail is shown", "Transmite a saída de cada check durante a execução; por padrão ela é capturada e o final da saída de um check que falha é exibido"},
		},
		Examples: []string{
			"pose validate --strict",
			"pose validate --module pose-mcp --verbose",
			"pose validate --module pose-mcp --strict",
			"pose validate --json-out .pose/results/delivery-validation.json",
		},
	},
	"check": {
		Name:            "check",
		SummaryEN:       "Verify structural integrity, matrix schema, task maps, and specs",
		SummaryPtBR:     "Verifica integridade estrutural, schema da matriz, task maps e specs",
		Usage:           "pose check [--strict|--tolerant] [--json] [--json-out <path>] [--quiet] [--color auto|always|never]",
		DescriptionEN:   "Performs comprehensive structural verification on all POSE files, broken markdown links, frontmatter syntax, matrix JSON schemas, and spec graphs.",
		DescriptionPtBR: "Realiza verificação estrutural completa em todos os arquivos do POSE, links quebrados em markdown, sintaxe de frontmatter, schemas JSON e grafos de specs.",
		Flags: []FlagHelp{
			{"--json", "Print one JSON document with the verdict, findings and counts instead of the human report", "Imprime um documento JSON com veredito, findings e contagens no lugar do relatório humano"},
			{"--json-out <path>", "Also write the JSON document to a project-relative file, keeping the human report", "Também grava o documento JSON num arquivo relativo ao projeto, mantendo o relatório humano"},
			{"--quiet", "Print the verdict alone", "Imprime apenas o veredito"},
			{"--color auto|always|never", "Force or suppress colour; NO_COLOR and POSE_COLOR are honoured", "Força ou suprime cor; NO_COLOR e POSE_COLOR são respeitados"},
			{"--strict", "Treat structural warnings as fatal validation failures", "Trata avisos estruturais como falhas fatais de validação"},
			{"--tolerant", "Report warnings without returning non-zero exit code", "Exibe avisos sem retornar código de saída diferente de zero"},
		},
		Examples: []string{
			"pose check",
			"pose check --strict",
		},
	},
	"lint-spec": {
		Name:            "lint-spec",
		SummaryEN:       "Lint specification lifecycle, sections, and requirement traceability",
		SummaryPtBR:     "Valida ciclo de vida, seções e rastreabilidade de requisitos da spec",
		Usage:           "pose lint-spec <slug>|--all [--strict|--tolerant] [--ready-check] [--required-only] [--design-check] [--json] [--json-out <path>] [--quiet] [--color auto|always|never]",
		DescriptionEN:   "Validates that a spec document conforms to the 7-section template, has stable requirement IDs (R1, R2), complete frontmatter, valid traceability evidence upon closeout, and an optional read-only design-basis projection.",
		DescriptionPtBR: "Valida se a spec está em conformidade com o template de 7 seções, IDs estáveis de requisitos (R1, R2), frontmatter completo, evidências válidas de rastreabilidade e uma projeção advisory opcional da base de decisões.",
		Flags: []FlagHelp{
			{"--ready-check", "Enforce Definition of Ready (DoR) gate before transitioning to in-progress", "Aplica o gate de Definition of Ready (DoR) antes de transicionar para in-progress"},
			{"--strict", "Fail on any missing requirement trace or unfilled section in done specs", "Falha em qualquer rastreio ausente ou seção não preenchida em specs concluídas"},
			{"--all", "Lint all specifications present under .pose/specs/", "Valida todas as especificações presentes sob .pose/specs/"},
			{"--required-only", "Check only mandatory core sections without optional decisions", "Verifica apenas seções obrigatórias sem decisões opcionais"},
			{"--design-check", "Project structured Assumption/Decision basis and objective diagnostics (advisory, read-only)", "Projeta base estruturada de premissas/decisões e diagnósticos objetivos (advisory, read-only)"},
			{"--json", "Print one JSON document with the verdict, findings and counts instead of the human report", "Imprime um documento JSON com veredito, findings e contagens no lugar do relatório humano"},
			{"--json-out <path>", "Also write the JSON document to a project-relative file, keeping the human report", "Também grava o documento JSON num arquivo relativo ao projeto, mantendo o relatório humano"},
			{"--quiet", "Print the verdict alone", "Imprime apenas o veredito"},
			{"--color auto|always|never", "Force or suppress colour; NO_COLOR and POSE_COLOR are honoured", "Força ou suprime cor; NO_COLOR e POSE_COLOR são respeitados"},
		},
		Examples: []string{
			"pose lint-spec my-feature --ready-check",
			"pose lint-spec my-feature --strict",
			"pose lint-spec --all --strict",
		},
	},
	"specs": {
		Name:            "specs",
		SummaryEN:       "List, filter, and discover specifications across the repository",
		SummaryPtBR:     "Lista, filtra e descobre especificações em todo o repositório",
		Usage:           "pose specs [--recent <N>] [--status <s>] [--since <d>] [--components <c>] [--json] | pose specs facts <slug> [--json]",
		DescriptionEN:   "Discovers and lists repository specifications sorted chronologically (newest first). Supports filtering by lifecycle status, tags, recent limit, and relative time windows.",
		DescriptionPtBR: "Descobre e lista especificações do repositório ordenadas cronologicamente (mais recentes primeiro). Suporta filtros por status, tags, limite recente e janelas de tempo.",
		Flags: []FlagHelp{
			{"--recent <N>", "Limit output to the N most recently created specifications", "Limita a saída às N especificações mais recentes"},
			{"--status <status>", "Filter by lifecycle status (draft, in-progress, done, blocked)", "Filtra pelo status do ciclo de vida (draft, in-progress, done, blocked)"},
			{"--since <window>", "Filter specs created within a relative duration (e.g. 14d, 30d) or ISO date", "Filtra specs criadas em uma janela relativa (ex: 14d, 30d) ou data ISO"},
			{"--components <tags>", "Filter by comma-separated component tags with OR semantics", "Filtra por tags de componentes separadas por vírgula"},
			{"--json", "Output specifications in structured JSON format", "Emite as especificações em formato JSON estruturado"},
		},
		Examples: []string{
			"pose specs --recent 10",
			"pose specs --status in-progress",
			"pose specs --since 14d",
			"pose specs --json",
		},
	},
	"spec-format": {
		Name:            "spec-format",
		SummaryEN:       "Inspect and migrate specifications to the modern chronological format",
		SummaryPtBR:     "Inspeciona e migra especificações para o formato cronológico moderno",
		Usage:           "pose spec-format <migrate|status> [<slug>|--all] [--format folder|flat] [--dry-run] [--json]",
		DescriptionEN:   "Migrates legacy spec structures to date-prefixed chronological layouts (YYYY-MM-DD-<slug>/spec.md or YYYY-MM-DD-<slug>.md), preserving companion files (amendments.jsonl) in directory envelopes.",
		DescriptionPtBR: "Migra estruturas legadas de specs para formatos cronológicos com prefixo de data, preservando arquivos acompanhantes (amendments.jsonl) em envelopes de diretório.",
		Subcommands: []SubcommandHelp{
			{
				Name:        "migrate",
				Usage:       "pose spec-format migrate <slug>|--all [--format folder|flat] [--dry-run] [--json]",
				SummaryEN:   "Migrate legacy specification structures into date-prefixed layouts (preserves companions)",
				SummaryPtBR: "Migra estruturas legadas de specs para layouts com prefixo de data (preserva acompanhantes)",
			},
			{
				Name:        "status",
				Usage:       "pose spec-format status [--json]",
				SummaryEN:   "Display current spec format conformance and count legacy structures",
				SummaryPtBR: "Exibe a conformidade de formato de specs e conta estruturas legadas",
			},
		},
		Flags: []FlagHelp{
			{"--all", "Target all specifications in repository", "Aplica a todas as especificações do repositório"},
			{"--format <folder|flat>", "Format preference: folder (default) or flat", "Preferência de formato: folder (padrão) ou flat"},
			{"--dry-run", "Preview migration actions without modifying disk", "Pré-visualiza ações de migração sem modificar o disco"},
			{"--json", "Output structured JSON report", "Emite relatório em JSON estruturado"},
		},
		Examples: []string{
			"pose spec-format status",
			"pose spec-format migrate my-feature --dry-run",
			"pose spec-format migrate --all",
		},
	},
	"spec-transfer": {
		Name:            "spec-transfer",
		SummaryEN:       "Preview and perform an authorized cross-project specification transfer",
		SummaryPtBR:     "Visualiza e executa uma transferência autorizada de spec entre projetos",
		Usage:           "pose spec-transfer <preview|apply|resume|status> [options]",
		DescriptionEN:   "Preview is read-only and emits a digest-bound plan. Apply and resume require a digest plus explicit --authorize-project values for every affected project. Status reads the journal for one selected project.",
		DescriptionPtBR: "Preview é somente leitura e emite um plano vinculado a digest. Apply e resume exigem digest e --authorize-project explícito para cada projeto afetado. Status lê o journal de um projeto selecionado.",
		Subcommands: []SubcommandHelp{
			{"preview", "pose spec-transfer preview --source <xref> --destination <xref> --map <source=disposition[:destination]> [--map-file <json>] [--mode reconcile-terminal] [--date YYYY-MM-DD]", "Create a deterministic read-only plan; reconcile-terminal retires a coordinator onto a done, closed executor without writing it", "Cria um plano determinístico somente leitura; reconcile-terminal retira um coordenador para um executor done e fechado sem escrevê-lo"},
			{"apply", "pose spec-transfer apply --plan <file> --digest <sha256> --authorize-project <id>...", "Apply a reviewed plan with per-project authorization", "Aplica um plano revisado com autorização por projeto"},
			{"resume", "pose spec-transfer resume --operation <id> --project <id> --authorize-project <id>...", "Resume an interrupted operation from its verified journal", "Retoma uma operação interrompida a partir do journal verificado"},
			{"status", "pose spec-transfer status --operation <id> [--project <id>]", "Read transfer phase for one project", "Lê a fase da transferência para um projeto"},
		},
		Examples: []string{
			"pose spec-transfer preview --source xref:proj.alpha/spec:work --destination xref:proj.beta/spec:work --map R1=equivalent:R1",
			"pose spec-transfer apply --plan transfer.json --digest <digest> --authorize-project proj.alpha --authorize-project proj.beta",
			"pose spec-transfer preview --mode reconcile-terminal --source xref:proj.alpha/spec:work --destination xref:proj.beta/spec:work --map-file map.json",
		},
	},
	"new-spec": {
		Name:            "new-spec",
		SummaryEN:       "Scaffold a new feature specification",
		SummaryPtBR:     "Cria o scaffold de uma nova especificação de feature",
		Usage:           "pose new-spec <slug> [--folder] [--legacy] [--surface minimal|standard|full] [--task <artifact-ref>] [--expect-context <digest>]",
		DescriptionEN:   "Creates a local draft spec, reuses a task's canonical spec when it already resolves, and routes an explicitly qualified cross-project create only when the configured project binding and context revision still match.",
		DescriptionPtBR: "Cria uma spec local em rascunho, reutiliza a spec canônica quando a tarefa já resolve e só roteia uma criação entre projetos quando o vínculo configurado e a revisão do contexto continuam válidos.",
		Examples: []string{
			"pose new-spec user-authentication",
			"pose new-spec billing-export-pipeline --folder",
		},
	},
	"new-spec-qualified": {
		Name:            "new-spec-qualified",
		SummaryEN:       "Create or reuse a spec through an explicit qualified authority",
		SummaryPtBR:     "Cria ou reutiliza uma spec pela autoridade qualificada explícita",
		Usage:           "pose new-spec-qualified <slug> --task xref:<project>/spec:<slug> --expect-context <digest>",
		DescriptionEN:   "Requires a qualified task and fresh context. The separate verb makes older engines reject the operation before they can ignore new flags on new-spec.",
		DescriptionPtBR: "Exige tarefa qualificada e contexto atual. O verbo separado faz motores antigos recusarem a operação antes de ignorarem opções novas de new-spec.",
		Examples: []string{
			"pose new-spec-qualified checkout --task xref:proj.executor/spec:checkout --expect-context <digest>",
		},
	},
	"new-roadmap": {
		Name:            "new-roadmap",
		SummaryEN:       "Scaffold a new governed roadmap in .pose/roadmaps/",
		SummaryPtBR:     "Cria um novo roadmap governado sob .pose/roadmaps/",
		Usage:           "pose new-roadmap <slug>",
		DescriptionEN:   "Creates a new roadmap definition artifact tracking milestones, DAG dependencies, planned delivery dates, and member specs.",
		DescriptionPtBR: "Cria um novo artefato de roadmap para rastrear marcos, dependências em DAG, datas planejadas e specs associadas.",
		Examples: []string{
			"pose new-roadmap platform-modernization",
		},
	},
	"new-adr": {
		Name:            "new-adr",
		SummaryEN:       "Create a new Architectural Decision Record (ADR)",
		SummaryPtBR:     "Cria um novo Registro de Decisão Arquitetural (ADR)",
		Usage:           "pose new-adr \"<title>\"",
		DescriptionEN:   "Creates a dated, numbered ADR under .pose/adr/ capturing architectural context, trade-offs, decision choices, and consequences.",
		DescriptionPtBR: "Cria um ADR datado e numerado sob .pose/adr/ registrando contexto arquitetural, trade-offs, decisões e consequências.",
		Examples: []string{
			"pose new-adr \"Adopt PostgreSQL for Transaction Storage\"",
		},
	},
	"new-knowledge": {
		Name:            "new-knowledge",
		SummaryEN:       "Create a governed knowledge artifact (handoff, note, decision-log)",
		SummaryPtBR:     "Cria um artefato de conhecimento governado (handoff, note, decision-log)",
		Usage:           "pose new-knowledge <handoff|note|decision-log> <slug> [--owner @alias] [--ttl-days N] [--restricted]",
		DescriptionEN:   "Scaffolds an operational knowledge file under .pose/knowledge/ with mandatory frontmatter, TTL retention expiration, and owner attribution.",
		DescriptionPtBR: "Cria um arquivo de conhecimento operacional sob .pose/knowledge/ com frontmatter obrigatório, prazo de TTL e atribuição de responsável.",
		Flags: []FlagHelp{
			{"--owner <@alias>", "Set the responsible owner alias for knowledge maintenance", "Define o alias do responsável pela manutenção do conhecimento"},
			{"--ttl-days <N>", "Set expiration TTL in days (max 90 days)", "Define o TTL de expiração em dias (máximo de 90 dias)"},
			{"--restricted", "Tag the knowledge item as restricted access", "Marca o item de conhecimento como de acesso restrito"},
		},
		Examples: []string{
			"pose new-knowledge handoff sprint-12-handoff --owner @techlead --ttl-days 14",
			"pose new-knowledge note kafka-partitioning-strategy --owner @backend",
		},
	},
	"review": {
		Name:            "review",
		SummaryEN:       "Governed review planning, bundle sealing, and attestation workflow",
		SummaryPtBR:     "Fluxo governado de planejamento de review, selagem de bundle e atestação",
		Usage:           "pose review <bundle|auto-attest|attest|verify|record|attribution-supplement> <scope|xref> [options]",
		DescriptionEN:   "Manages component-aware reviews and immutable bundles. Cross-project scope writes require an explicit project binding and a fresh --expect-context digest.",
		DescriptionPtBR: "Gerencia reviews por componentes e bundles imutáveis. Escritas entre projetos exigem vínculo explícito e digest --expect-context atual.",
		Subcommands: []SubcommandHelp{
			{"bundle", "pose review bundle <scope|xref> [--seal] [--explain] [--expect-context <digest>]", "Prepare or seal an immutable review subject bundle", "Prepara ou sela o bundle imutável de revisão"},
			{"auto-attest", "pose review auto-attest <scope|bundle-id> [--apply]", "Extract validation results and automatically record attestation", "Extrai resultados de validação e registra atestação automaticamente"},
			{"attest", "pose review attest <bundle-id|scope|xref> --reviewer <id> --decision <decision> --evidence <ref> [--criterion ID|disposition|evidence|rationale] [--mapping CRITERION|DELTA|basis-or-disposition|rationale] [--prepared-by <p>] [--concluded-by <p>] [--confirmed-by <p> --confirmation-mode adopted-conclusions|authorized-operation] [--applied-by <p>] [--apply] [--expect-context <digest>]", "Record a manual review decision attestation; attribution roles are explicit, never derived from --reviewer, and a confirmation is bound to the exact content written; cross-project writes may target a qualified xref scope only with a fresh context digest", "Registra uma atestação manual; os papéis de atribuição são explícitos, nunca derivados de --reviewer, e uma confirmação fica vinculada ao conteúdo exato gravado; escritas externas podem usar escopo xref qualificado somente com digest de contexto atual"},
			{"attribution-supplement", "pose review attribution-supplement <attestation-id> --recorded-by <p> --note <text> [--prepared-by <p>] [--concluded-by <p>] [--applied-by <p>] [--evidence <ref>] [--apply]", "Clarify who prepared, concluded and applied an existing attestation without altering it; a supplement cannot add a confirmation", "Esclarece quem preparou, concluiu e aplicou uma atestação existente sem alterá-la; um suplemento não pode acrescentar confirmação"},
			{"verify", "pose review verify <scope|bundle-id>", "Verify freshness and closeout readiness of review bundles", "Verifica atualidade e prontidão do review bundle para fechamento"},
			{"record", "pose review record <scope|xref> ... [--expect-context <digest>]", "Compatibility entrypoint for review recording", "Ponto de entrada de compatibilidade para registro de review"},
		},
		Examples: []string{
			"pose review bundle spec:my-feature --seal",
			"pose review auto-attest spec:my-feature --apply",
			"pose review verify spec:my-feature",
		},
	},
	"action": {
		Name:            "action",
		SummaryEN:       "Open, read and resolve material requests to a person or an external system",
		SummaryPtBR:     "Abre, lê e resolve solicitações materiais a uma pessoa ou a um sistema externo",
		Usage:           "pose action <open|list|show|resolve|cancel|waive|invalidate> ... | pose action list --present [--actor <id|role>] [--json]",
		DescriptionEN:   "An action request records a decision, approval, input, external operation or acceptance that no other source holds, with its question, options and consequences, recipient, qualified targets and per-phase effects. Open one only when a different answer would materially change execution, scope, authority, risk acceptance, closeout or publication, and no authorization already given covers it. Writes are previews until --apply. A resolution names the request digest it answers and the revision it read, carries an idempotency key, and is accepted only from an actor holding the recipient role in .pose/policy/actions.json; under verified assurance it also needs a signed claim. Answered is not satisfied: a declined approval, a rejected acceptance or a failed operation leaves the request's condition unmet.",
		DescriptionPtBR: "Uma action request registra uma decisão, aprovação, informação, operação externa ou aceite que nenhuma outra fonte guarda, com a pergunta, as opções e consequências, o destinatário, os alvos qualificados e os efeitos por fase. Abra uma somente quando uma resposta diferente mudaria materialmente execução, escopo, autoridade, aceitação de risco, closeout ou publicação, e nenhuma autorização já dada a cobre. Escritas são preview até --apply. Uma resolução nomeia o digest da solicitação que responde e a revisão que leu, traz uma chave de idempotência e só é aceita de um ator que detém o papel destinatário em .pose/policy/actions.json; com garantia verificada também exige uma claim assinada. Respondida não é satisfeita: uma aprovação recusada, um aceite rejeitado ou uma operação falha deixam a condição sem atendimento.",
		Subcommands: []SubcommandHelp{
			{"open", "pose action open --origin <ref> --kind <kind> --question <text> --requested-by <p> --target <ref>... --effect <phase>:<mode>... [--option id=consequence]... [--recipient <p>|--recipient-role <r>] [--subject <ref|path:file>] [--supersedes <id>] [--apply]", "Preview or record a material request", "Visualiza ou registra uma solicitação material"},
			{"list", "pose action list [--state <state>] [--json]", "List requests with their derived state", "Lista solicitações com o estado derivado"},
			{"show", "pose action show <act-id> [--json]", "Show exactly the content an answer is bound to", "Mostra exatamente o conteúdo ao qual uma resposta fica vinculada"},
			{"resolve", "pose action resolve <act-id> --actor <p> --answer <a> --request-digest <d> --expected-revision <n> --idempotency-key <k> [--claim <file>] [--apply]", "Record an answer from an authorized actor", "Registra a resposta de um ator autorizado"},
			{"cancel", "pose action cancel <act-id> --actor <p> --reason <text> --request-digest <d> --expected-revision <n> --idempotency-key <k> [--apply]", "Withdraw a request (requester or authority)", "Retira uma solicitação (solicitante ou autoridade)"},
			{"waive", "pose action waive <act-id> --actor <p> --reason <text> --request-digest <d> --expected-revision <n> --idempotency-key <k> [--apply]", "Dispense with the condition (authority only)", "Dispensa a condição (somente a autoridade)"},
			{"invalidate", "pose action invalidate <act-id> --actor <p> --reason <text> --request-digest <d> --expected-revision <n> --idempotency-key <k> [--apply]", "Mark an answer as no longer applying", "Marca uma resposta como não aplicável"},
		},
		Examples: []string{
			"pose action open --origin spec:storage --kind decision --question \"Keep reading schema v1?\" --option preserve-v1=\"Keep the v1 reader\" --option break-v1=\"Consumers migrate\" --recipient-role maintainer --requested-by agent:impl --target requirement:R4 --effect execution:block --effect closeout:block --apply",
			"pose action resolve act-0123456789abcdef --actor human:maintainer --answer preserve-v1 --request-digest sha256:... --expected-revision 1 --idempotency-key answer-1 --apply",
		},
	},
	"adopt": {
		Name:            "adopt",
		SummaryEN:       "Turn a governed capability on or off in this instance",
		SummaryPtBR:     "Liga ou desliga uma capacidade governada nesta instância",
		Usage:           "pose adopt --list [--json] | pose adopt <capability> [--off | --decline --reason <text> | --defer --reason <text>] [--date YYYY-MM-DD] [--apply]",
		DescriptionEN:   "Decides one capability from the catalog — every review-policy capability, the review overlays, the Definition of Ready, qualified references and spec authority transfer. Without a flag it previews the keys that turn it on, and --apply writes them, dated today or --date so work created earlier is not re-judged; adopting an adopted capability keeps its date. --off removes them, refused while another adopted capability depends on it. --decline and --defer record in .pose/policy/adoption-decisions.json that the project chose, for a reason, not to adopt it now. A capability whose requirement or prerequisite is missing is refused with what is missing. Every write is refused when the policy readers would refuse it. --list shows every capability with its state: on, off, declined, deferred or needs-setup. A new instance adopts the catalog's defaults at install; pose update never adopts.",
		DescriptionPtBR: "Decide uma capacidade do catálogo — toda capacidade da policy de review, os overlays de review, a Definition of Ready, referências qualificadas e transferência de autoridade de spec. Sem flag, mostra as chaves que a ligam, e --apply as grava, datadas com hoje ou --date para que trabalho criado antes não seja rejulgado; adotar uma capacidade já adotada mantém a data. --off as remove, recusado enquanto outra capacidade adotada depender dela. --decline e --defer registram em .pose/policy/adoption-decisions.json que o projeto decidiu, por um motivo, não adotá-la agora. Capacidade com requisito ou pré-requisito faltando é recusada com o que falta. Toda escrita é recusada quando os leitores da policy a recusariam. --list mostra cada capacidade com seu estado: on, off, declined, deferred ou needs-setup. Uma instância nova adota os padrões do catálogo na instalação; pose update nunca adota.",
		Flags: []FlagHelp{
			{"--list", "Print every capability in the catalog with its state", "Mostra cada capacidade do catálogo com seu estado"},
			{"--json", "With --list, print the catalog as JSON", "Com --list, mostra o catálogo em JSON"},
			{"--off", "Remove the capability's keys", "Remove as chaves da capacidade"},
			{"--decline", "Record that this project chose not to adopt it (needs --reason)", "Registra que este projeto decidiu não adotar (exige --reason)"},
			{"--defer", "Record that this project will decide later (needs --reason)", "Registra que este projeto vai decidir depois (exige --reason)"},
			{"--reason <text>", "Why a capability was declined or deferred", "Por que a capacidade foi recusada ou adiada"},
			{"--date YYYY-MM-DD", "Adoption date for a capability with a cutoff (default today)", "Data de adoção para capacidade com corte (padrão hoje)"},
			{"--apply", "Write the previewed change", "Grava a mudança mostrada"},
		},
		Examples: []string{
			"pose adopt --list",
			"pose adopt causality-closeout",
			"pose adopt overlay:engineering-judgment --decline --reason \"judgment is reviewed by people here\" --apply",
			"pose adopt atomic-start --date 2026-10-06 --apply",
			"pose adopt contract-nodes --off --apply",
		},
	},
	"start": {
		Name:            "start",
		SummaryEN:       "Preview or apply the atomic start of a draft spec",
		SummaryPtBR:     "Visualiza ou aplica o início atômico de uma spec draft",
		Usage:           "pose start spec:<slug> [--json] | --apply --digest <sha256> | --status [--json] | --cancel",
		DescriptionEN:   "Preview is read-only: readiness, dependencies, declared review obligations and the R/A/D baseline in a digest-bound plan. --apply records that baseline and moves the spec from draft to in-progress under a per-spec lock; it needs atomic_start_version in the review policy and is idempotent and resumable. --status classifies nodes as recorded before managed execution, introduced during it, or legacy-unbaselined. A start records observable precedence, never when anything was decided.",
		DescriptionPtBR: "Preview é somente leitura: readiness, dependências, obrigações de review declaradas e a baseline R/A/D num plano vinculado a digest. --apply grava essa baseline e move a spec de draft para in-progress com lock por spec; exige atomic_start_version na policy de review e é idempotente e retomável. --status classifica os nós como registrados antes da execução gerenciada, introduzidos durante ela ou legacy-unbaselined. Um start registra precedência observável, nunca quando algo foi decidido.",
		Flags: []FlagHelp{
			{"--json", "Emit the plan, record or status as JSON", "Emite o plano, o registro ou o status em JSON"},
			{"--apply", "Apply the reviewed preview", "Aplica o preview revisado"},
			{"--digest <sha256>", "Digest of the reviewed preview", "Digest do preview revisado"},
			{"--status", "Classify nodes and list reconciliation needs", "Classifica nós e lista reconciliações"},
			{"--cancel", "Undo a start that recorded its baseline but never moved the spec", "Desfaz um start que gravou a baseline sem mover a spec"},
		},
		Examples: []string{
			"pose start spec:user-authentication",
			"pose start spec:user-authentication --apply --digest <sha256>",
		},
	},
	"close": {
		Name:            "close",
		SummaryEN:       "Apply review-gated lifecycle closeout to a spec or milestone",
		SummaryPtBR:     "Aplica fechamento governado com portão de review em spec ou marco",
		Usage:           "pose close <scope|xref> [--expect-context <digest>] [--json] | pose close spec:<slug> --plan|--apply|--resume [--reviewer agent:<id>] [--digest <plan digest>]",
		DescriptionEN:   "Transitions a completed specification (or milestone) after verifying review and delivery gates. Cross-project close requires an explicit project binding and fresh context digest.",
		DescriptionPtBR: "Transiciona uma spec (ou marco) para status: done após verificar se atestações de review, rastreio de requisitos e garantias de entrega foram cumpridos.",
		Flags: []FlagHelp{
			{"--json", "Output closeout transition details in JSON format", "Emite os detalhes da transição de fechamento em formato JSON"},
			{"--plan", "Preview the ordered, recoverable closeout plan of a spec", "Mostra o plano ordenado e recuperável de closeout de uma spec"},
			{"--apply", "Run the plan's mechanical steps; stop at a pending judgment (exit 3)", "Executa os passos mecânicos do plano; para num julgamento pendente (saída 3)"},
			{"--resume", "Re-plan after an interruption and continue without repeating done steps", "Recalcula o plano após uma interrupção e continua sem repetir passos concluídos"},
			{"--reviewer <id>", "Reviewing execution that records the mechanical attestation", "Execução revisora que registra a attestation mecânica"},
			{"--expect-context <digest>", "Require the context revision returned by a fresh pose context call", "Exige a revisão de contexto retornada por uma chamada pose context atual"},
		},
		Examples: []string{
			"pose close spec:user-authentication",
			"pose close milestone:roadmap-slug/m1",
		},
	},
	"extension": {
		Name:            "extension",
		SummaryEN:       "Manage POSE modular domain extensions (rules, skills, workflows)",
		SummaryPtBR:     "Gerencia extensões modulares de domínio do POSE (regras, skills, fluxos)",
		Usage:           "pose extension <install|list|remove|verify> [options]",
		DescriptionEN:   "Installs, verifies, lists, and uninstalls modular POSE extensions from local directories, registries, or repository packages.",
		DescriptionPtBR: "Instala, verifica, lista e desinstala extensões modulares do POSE a partir de diretórios locais, registros ou pacotes.",
		Subcommands: []SubcommandHelp{
			{"install", "pose extension install <dir> [--target <dir>] [--allow-unsigned]", "Install a modular extension into the project", "Instala uma extensão modular no projeto"},
			{"list", "pose extension list [--json]", "List all currently installed extensions in .pose/policy/extensions.lock.json", "Lista todas as extensões instaladas no projeto"},
			{"remove", "pose extension remove <id>", "Uninstall and remove an extension's installed files", "Desinstala e remove os arquivos de uma extensão"},
			{"verify", "pose extension verify <dir> [--allow-unsigned]", "Verify extension manifest schema, file digests, and signatures", "Verifica o manifesto, hashes e assinaturas de uma extensão"},
		},
		Examples: []string{
			"pose extension install extensions/pose-rule-backend-go",
			"pose extension list",
			"pose extension verify extensions/pose-rule-frontend-vue --allow-unsigned",
		},
	},
	"contribute": {
		Name:            "contribute",
		SummaryEN:       "Manage Open-Source POSE Contributor Mode and feedback staging",
		SummaryPtBR:     "Gerencia o Modo Contribuidor Open-Source do POSE e rascunhos de feedback",
		Usage:           "pose contribute <enable|disable|status|stage|list> [--target <dir>] [--json]",
		DescriptionEN:   "Controls POSE Contributor Mode; agents ask for user confirmation before staging sanitized feedback under .pose/contributions/ or submitting it upstream.",
		DescriptionPtBR: "Controla o Modo Contribuidor do POSE; agentes solicitam confirmação do usuário antes de registrar feedback sanitizado sob .pose/contributions/ ou submetê-lo upstream.",
		Subcommands: []SubcommandHelp{
			{"enable", "pose contribute enable [--target <dir>]", "Enable contributor mode and inject governed agent instructions", "Ativa o modo contribuidor e injeta instruções governadas de agente"},
			{"disable", "pose contribute disable [--target <dir>]", "Disable contributor mode and remove instructions from manuals", "Desativa o modo contribuidor e remove instruções dos manuais"},
			{"status", "pose contribute status [--json]", "Display active contributor status, count of staged drafts, and privacy rules", "Exibe o status do modo contribuidor, contagem de rascunhos e regras de privacidade"},
			{"stage", "pose contribute stage --title \"...\" [--type bug|enhancement] [--body \"...\"]", "Record a structured feedback proposal locally", "Registra formalmente uma proposta de feedback em rascunho"},
			{"list", "pose contribute list [--json]", "List all staged contributions awaiting developer adjudication", "Lista todos os rascunhos locais aguardando avaliação do desenvolvedor"},
		},
		Examples: []string{
			"pose contribute enable",
			"pose contribute status",
			"pose contribute stage --title \"Missing Svelte 5 Rune check\" --type enhancement",
			"pose contribute list",
		},
	},
	"release": {
		Name:            "release",
		SummaryEN:       "Governed release planning, freezing, verification, and publication",
		SummaryPtBR:     "Planejamento, congelamento, verificação e publicação governada de releases",
		Usage:           "pose release <plan|prepare|check|notes|record|status|open-next|backfill> --version vX.Y.Z",
		DescriptionEN:   "Orchestrates immutable release cycles, compiling changelog fragments, generating release notes, and validating delivery assurance.",
		DescriptionPtBR: "Orquestra ciclos imutáveis de release, compilando fragmentos de changelog, gerando notas de release e validando garantias de entrega.",
		Subcommands: []SubcommandHelp{
			{"plan", "pose release plan --version vX.Y.Z", "Preview the release cut, eligible specs, and changelog entries", "Visualiza o corte de release, specs elegíveis e itens do changelog"},
			{"prepare", "pose release prepare --version vX.Y.Z [--apply]", "Freeze selected fragments into canonical release notes and manifest", "Congela fragmentos em notas canônicas de release e manifesto"},
			{"check", "pose release check --version vX.Y.Z", "Validate release readiness and delivery evidence completeness", "Valida prontidão da release e completude das evidências de entrega"},
			{"notes", "pose release notes --version vX.Y.Z", "Display the immutable release notes for the specified version", "Exibe as notas imutáveis de release da versão especificada"},
			{"record", "pose release record --version vX.Y.Z --event tagged|published|verified|failed|yanked --evidence <file>", "Import provider or verification evidence as an append-only release event", "Importa evidência do provedor ou da verificação como evento de release append-only"},
			{"status", "pose release status", "Display the current release lifecycle pipeline state", "Exibe o estado atual do pipeline de release"},
		},
		Examples: []string{
			"pose release plan --version v1.5.0",
			"pose release prepare --version v1.5.0 --apply",
			"pose release check --version v1.5.0",
		},
	},
	"state": {
		Name:            "state",
		SummaryEN:       "Manage native project state artifact",
		SummaryPtBR:     "Gerencia o artefato nativo de estado do projeto",
		Usage:           "pose state [--json] [--json-out <path>] [--quiet] [--color auto|always|never] | pose state [init|refresh [--if-stale]|diff]",
		DescriptionEN:   "Generates, updates, and diffs the native .pose/state/project-state.json artifact capturing specs, debts, and repository health.",
		DescriptionPtBR: "Gera, atualiza e compara o artefato nativo .pose/state/project-state.json com specs, débitos e integridade do repositório.",
		Flags: []FlagHelp{
			{"--if-stale", "Refresh state only if the existing artifact exceeds the staleness threshold", "Atualiza o estado apenas se o artefato existente ultrapassar o limite de desatualização"},
			{"--json", "Print one JSON document with the verdict, findings and counts instead of the human report", "Imprime um documento JSON com veredito, findings e contagens no lugar do relatório humano"},
			{"--json-out <path>", "Also write the JSON document to a project-relative file, keeping the human report", "Também grava o documento JSON num arquivo relativo ao projeto, mantendo o relatório humano"},
			{"--quiet", "Print the verdict alone", "Imprime apenas o veredito"},
			{"--color auto|always|never", "Force or suppress colour; NO_COLOR and POSE_COLOR are honoured", "Força ou suprime cor; NO_COLOR e POSE_COLOR são respeitados"},
		},
		Examples: []string{
			"pose state",
			"pose state refresh --if-stale",
			"pose state diff",
		},
	},
	"followups": {
		Name:            "followups",
		SummaryEN:       "List and triage open spec follow-ups and near-duplicates",
		SummaryPtBR:     "Lista e faz triagem de follow-ups em aberto e quase-duplicatas",
		Usage:           "pose followups [--open|--all] [--json] [--overdue] | pose followups --candidates [--json]",
		DescriptionEN:   "Aggregates all follow-up items declared across specs in section 7 (Final Report), flagging overdue SLAs and near-duplicate proposals.",
		DescriptionPtBR: "Agrega todos os follow-ups declarados nas specs na seção 7 (Final Report), sinalizando SLAs vencidos e propostas quase-duplicadas.",
		Flags: []FlagHelp{
			{"--open", "List only open, untriaged follow-ups (default)", "Lista apenas follow-ups em aberto e não triados (padrão)"},
			{"--all", "List all follow-ups across all specs regardless of disposition", "Lista todos os follow-ups em todas as specs independentemente da disposição"},
			{"--overdue", "Filter to follow-ups exceeding their review date SLA", "Filtra follow-ups que ultrapassaram a data limite de revisão"},
			{"--json", "Output follow-ups in structured JSON format", "Emite os follow-ups em formato JSON estruturado"},
		},
		Examples: []string{
			"pose followups --open",
			"pose followups --overdue",
		},
	},
	"index": {
		Name:            "index",
		SummaryEN:       "Regenerate POSE cached indexes (.pose/indexes/)",
		SummaryPtBR:     "Regenera os índices cacheados do POSE (.pose/indexes/)",
		Usage:           "pose index [--json] [--json-out <path>] [--quiet] [--color auto|always|never]",
		DescriptionEN:   "Recomputes and writes all static indexes including spec-graph.json, roadmaps.json, releases.json, and delivery-integrity.json.",
		DescriptionPtBR: "Recalcula e grava todos os índices estáticos incluindo spec-graph.json, roadmaps.json, releases.json e delivery-integrity.json.",
		Flags: []FlagHelp{
			{"--json", "Print one JSON document with the verdict, findings and counts instead of the human report", "Imprime um documento JSON com veredito, findings e contagens no lugar do relatório humano"},
			{"--json-out <path>", "Also write the JSON document to a project-relative file, keeping the human report", "Também grava o documento JSON num arquivo relativo ao projeto, mantendo o relatório humano"},
			{"--quiet", "Print the verdict alone", "Imprime apenas o veredito"},
			{"--color auto|always|never", "Force or suppress colour; NO_COLOR and POSE_COLOR are honoured", "Força ou suprime cor; NO_COLOR e POSE_COLOR são respeitados"},
		},
		Examples: []string{
			"pose index",
		},
	},
	"update": {
		Name:            "update",
		SummaryEN:       "Update instance contracts and machinery to match engine version",
		SummaryPtBR:     "Atualiza contratos e maquinário da instância para a versão do motor",
		Usage:           "pose update [--dry-run] [--force] [--no-self] [--locale <tag>]",
		DescriptionEN:   "Applies idempotent schema migrations and merges canonical workflow, rule, and manual updates while preserving instance-owned and contributor sections.",
		DescriptionPtBR: "Aplica migrações idempotentes de schema e mescla atualizações de manuais, workflows e regras, preservando seções da instância e do modo contribuidor.",
		Flags: []FlagHelp{
			{"--dry-run", "Preview file merges and migrations without writing changes", "Visualiza as mesclagens e migrações sem gravar alterações"},
			{"--force", "Also rerun install over the instance: reset managed manuals wholesale and refresh scaffolds, rules, workflows and MCP config", "Também reexecuta o install sobre a instância: redefine os manuais por completo e atualiza scaffolds, regras, workflows e configuração MCP"},
			{"--no-self", "Skip the binary self-update and migrate the instance with the running binary", "Pula a autoatualização do binário e migra a instância com o binário em execução"},
			{"--locale <tag>", "Locale for managed manuals and machinery (default: the instance's own)", "Locale dos manuais gerenciados e da maquinaria (padrão: o da própria instância)"},
		},
		Examples: []string{
			"pose update",
			"pose update --dry-run",
		},
	},
	"hooks": {
		Name:            "hooks",
		SummaryEN:       "Manage POSE Git hooks (pre-commit gate, post-merge indexer)",
		SummaryPtBR:     "Gerencia git hooks do POSE (pre-commit gate, post-merge indexer)",
		Usage:           "pose hooks <install|uninstall|status>",
		DescriptionEN:   "Installs or removes deterministic git hooks under .git/hooks/ for automatic pre-commit structural validation and post-merge index recomputation.",
		DescriptionPtBR: "Instala ou remove hooks git determinísticos sob .git/hooks/ para validação estrutural automática no pre-commit e reindexação no post-merge.",
		Subcommands: []SubcommandHelp{
			{"install", "pose hooks install", "Install POSE pre-commit and post-merge git hooks", "Instala os git hooks de pre-commit e post-merge do POSE"},
			{"uninstall", "pose hooks uninstall", "Remove installed POSE git hooks", "Remove os git hooks instalados do POSE"},
			{"status", "pose hooks status", "Display the current installation status of git hooks", "Exibe o status atual de instalação dos git hooks"},
		},
		Examples: []string{
			"pose hooks install",
			"pose hooks status",
		},
	},
	"suggest": {
		Name:            "suggest",
		SummaryEN:       "Suggest relevant skills, workflows, and rules for a task",
		SummaryPtBR:     "Sugere skills, workflows e regras relevantes para uma tarefa",
		Usage:           "pose suggest [<task-type>] [--domain <d>] [--path <dir>] [--json]",
		DescriptionEN:   "Evaluates task intent and affected directories to recommend the exact minimal skill and domain rule extension without cognitive overload.",
		DescriptionPtBR: "Avalia a intenção da tarefa e diretórios afetados para recomendar a skill e extensão de regra exata sem sobrecarga cognitiva.",
		Flags: []FlagHelp{
			{"--domain <name>", "Filter suggestions to a specific engineering domain", "Filtra sugestões para um domínio específico de engenharia"},
			{"--path <dir>", "Analyze module stack at the given directory path", "Analisa a stack do módulo no caminho de diretório informado"},
			{"--json", "Output suggestions in structured JSON format", "Emite as sugestões em formato JSON estruturado"},
		},
		Examples: []string{
			"pose suggest feature --path pose-mcp",
			"pose suggest bugfix",
		},
	},
	"assess": {
		Name:            "assess",
		SummaryEN:       "Discover module metrics, integration contracts, and technical debt",
		SummaryPtBR:     "Descobre métricas de módulos, contratos de integração e débito técnico",
		Usage:           "pose assess <discover|integrate|tech-debt|design> [options]",
		DescriptionEN:   "Runs bounded local assessments for component LOC, dependencies, technical debt markers, public contracts, and structural deltas on a canonical review subject.",
		DescriptionPtBR: "Executa assessments locais e bounded de LOC, dependências, débito técnico, contratos públicos e deltas estruturais sobre um subject canônico de review.",
		Subcommands: []SubcommandHelp{
			{"discover", "pose assess discover [--component <dir>] [--update-state]", "Discover LOC metrics, debts, and module structure", "Descobre métricas de LOC, débitos e estrutura de módulos"},
			{"integrate", "pose assess integrate", "Check inter-module contracts (REST, Protobuf, Kafka, MCP)", "Verifica contratos inter-módulos (REST, Protobuf, Kafka, MCP)"},
			{"tech-debt", "pose assess tech-debt", "Scan codebase for technical debt markers and uncovered stubs", "Escaneia a base de código por marcadores de débito e stubs"},
			{"design", "pose assess design --spec <slug> [--json] [--max-files N] [--max-bytes N]", "Observe bounded structural deltas from the sealed review subject", "Observa deltas estruturais bounded a partir do subject selado de review"},
		},
		Examples: []string{
			"pose assess discover --update-state",
			"pose assess tech-debt",
			"pose assess design --spec customer-cache --json",
		},
	},
	"serve-mcp": {
		Name:            "serve-mcp",
		SummaryEN:       "Start the POSE Model Context Protocol (MCP) server",
		SummaryPtBR:     "Inicia o servidor MCP (Model Context Protocol) do POSE",
		Usage:           "pose serve-mcp [--stdio]",
		DescriptionEN:   "Runs the native POSE MCP server over standard input/output (stdio) or HTTP transport, exposing 20+ specialized tools to AI coding assistants.",
		DescriptionPtBR: "Executa o servidor nativo MCP do POSE via stdio ou HTTP, expondo mais de 20 ferramentas especializadas para assistentes de IA.",
		Flags: []FlagHelp{
			{"--stdio", "Run MCP server in stdio mode (default when managed by Claude Code / Cursor / Windsurf)", "Executa o servidor MCP via stdio (padrão quando gerenciado pelo cliente)"},
		},
		Examples: []string{
			"pose serve-mcp --stdio",
		},
	},
	"telemetry": {
		Name:            "telemetry",
		SummaryEN:       "Manage anonymous opt-in telemetry configuration",
		SummaryPtBR:     "Gerencia a configuração de telemetria anônima opt-in",
		Usage:           "pose telemetry <enable|disable|status>",
		DescriptionEN:   "Configures local anonymous telemetry. By default, telemetry is strictly disabled and transmits zero data without explicit opt-in.",
		DescriptionPtBR: "Configura a telemetria anônima local. Por padrão, a telemetria é estritamente desativada e transmite zero dados sem consentimento explícito.",
		Subcommands: []SubcommandHelp{
			{"enable", "pose telemetry enable", "Enable anonymous opt-in usage telemetry", "Ativa a telemetria anônima de uso opt-in"},
			{"disable", "pose telemetry disable", "Disable anonymous telemetry", "Desativa a telemetria anônima"},
			{"status", "pose telemetry status", "Display telemetry status and configured endpoint", "Exibe o status da telemetria e endpoint configurado"},
		},
		Examples: []string{
			"pose telemetry status",
			"pose telemetry enable",
		},
	},
	"import": {
		Name:            "import",
		SummaryEN:       "Import external SDD specifications into POSE format",
		SummaryPtBR:     "Importa especificações SDD externas para o formato POSE",
		Usage:           "pose import <spec-kit|openspec> <path> [--dry-run]",
		DescriptionEN:   "Converts external spec-kit feature trees or OpenSpec changes/proposals into native POSE specifications with preserved requirements.",
		DescriptionPtBR: "Converte árvores de features do spec-kit ou propostas do OpenSpec para especificações nativas do POSE com requisitos preservados.",
		Flags: []FlagHelp{
			{"--dry-run", "Preview imported specs without writing files to disk", "Visualiza as specs importadas sem gravar arquivos no disco"},
		},
		Examples: []string{
			"pose import spec-kit .specify/specs --dry-run",
			"pose import openspec openspec/changes/my-feature",
		},
	},
	"report-limitation": {
		Name:            "report-limitation",
		SummaryEN:       "Report a POSE engine defect, limitation, or feature proposal",
		SummaryPtBR:     "Relata um defeito, limitação ou proposta para o motor POSE",
		Usage:           "pose report-limitation --title \"<title>\" [--kind limitation|bug|suggestion] [--body \"<text>\"] [--submit]",
		DescriptionEN:   "Records a structured feedback artifact under .pose/feedback/. When --submit is passed and POSE_TELEMETRY_URL is set, submits the sanitized report upstream.",
		DescriptionPtBR: "Registra um artefato estruturado de feedback sob .pose/feedback/. Quando --submit é passado e POSE_TELEMETRY_URL está configurado, envia o relatório sanitizado.",
		Flags: []FlagHelp{
			{"--title <text>", "Short summary title of the observed limitation or bug", "Título descritivo da limitação ou bug observado"},
			{"--kind <type>", "Classification: limitation, bug, or suggestion (default: limitation)", "Classificação: limitation, bug ou suggestion (padrão: limitation)"},
			{"--body <text>", "Detailed reproduction steps, observed behavior, and proposed fix", "Passos de reprodução, comportamento observado e proposta de correção"},
			{"--submit", "Submit feedback upstream if telemetry endpoint is configured", "Envia o feedback para o repositório upstream se configurado"},
		},
		Examples: []string{
			"pose report-limitation --title \"Doctor fails to recognize pnpm workspace\" --kind bug",
		},
	},
	"artifact-check": {
		Name:            "artifact-check",
		SummaryEN:       "Reconcile declared spec artifacts with immutable Git changesets",
		SummaryPtBR:     "Reconcilia artefatos declarados na spec com o changeset imutável do Git",
		Usage:           "pose artifact-check --spec <slug> [--from <rev> --to <rev>] [--strict|--tolerant] [--json]",
		DescriptionEN:   "Verifies that all files declared in section 3 (Technical Plan -> Artifacts) match the actual Git change set attributed to the spec trailer (POSE-Spec: <slug>).",
		DescriptionPtBR: "Verifica se todos os arquivos declarados na seção 3 (Technical Plan -> Artifacts) correspondem ao changeset real do Git atribuído via trailer (POSE-Spec: <slug>).",
		Flags: []FlagHelp{
			{"--spec <slug>", "Specification slug to reconcile", "Slug da especificação a ser reconciliada"},
			{"--from <rev>", "Starting git revision for change set comparison", "Revisão git inicial para comparação do changeset"},
			{"--to <rev>", "Ending git revision for change set comparison", "Revisão git final para comparação do changeset"},
			{"--strict", "Fail on any undeclared or missing artifact mismatch", "Falha em qualquer discrepância de artefato não declarado ou ausente"},
			{"--json", "Output reconciliation results in JSON format", "Emite o resultado da reconciliação em formato JSON"},
		},
		Examples: []string{
			"pose artifact-check --spec user-auth --strict",
		},
	},
	"surface-check": {
		Name:            "surface-check",
		SummaryEN:       "Prove composition and reachability of delivered UI surfaces and capabilities",
		SummaryPtBR:     "Comprova composição e alcançabilidade de superfícies de UI e capacidades entregues",
		Usage:           "pose surface-check [--spec <slug>] [--results <path>] [--strict|--tolerant] [--json]",
		DescriptionEN:   "Validates that delivered surfaces, contracts, and capabilities are reachable from production entrypoints with fresh typed validation evidence.",
		DescriptionPtBR: "Valida se superfícies, contratos e capacidades entregues são alcançáveis a partir de pontos de entrada de produção com evidências atuais de validação.",
		Flags: []FlagHelp{
			{"--spec <slug>", "Filter check to delivery targets of a specific spec", "Filtra a checagem para os alvos de entrega de uma spec específica"},
			{"--results <path>", "Path to validation results JSON (default: .pose/results/delivery-validation.json)", "Caminho do JSON de resultados de validação"},
			{"--strict", "Fail on unreached surfaces or missing validation evidence", "Falha em superfícies inalcançáveis ou evidências ausentes"},
		},
		Examples: []string{
			"pose surface-check --spec dashboard-ui --strict",
		},
	},
	"roadmap-check": {
		Name:            "roadmap-check",
		SummaryEN:       "Evaluate roadmap milestones, cut criteria, and member spec closure",
		SummaryPtBR:     "Avalia marcos de roadmap, critérios de corte e fechamento de specs membros",
		Usage:           "pose roadmap-check <slug> [--strict|--tolerant] [--json]",
		DescriptionEN:   "Evaluates whether all member specs in a milestone or roadmap are sealed, attested, and meet release cut criteria.",
		DescriptionPtBR: "Avalia se todas as specs membros de um marco ou roadmap estão seladas, atestadas e atendem aos critérios de corte da release.",
		Flags: []FlagHelp{
			{"--strict", "Fail if any dependent spec is incomplete or unverified", "Falha se qualquer spec dependente estiver incompleta ou não verificada"},
			{"--json", "Output roadmap evaluation in JSON format", "Emite o relatório de avaliação do roadmap em JSON"},
		},
		Examples: []string{
			"pose roadmap-check platform-v2 --strict",
		},
	},
	"migrate": {
		Name:            "migrate",
		SummaryEN:       "Dry-run the legacy inventory a future major would convert, keep or refuse",
		SummaryPtBR:     "Executa o dry-run do inventário de legado que uma futura major converteria, manteria ou recusaria",
		Usage:           "pose migrate v7 --dry-run [--json]",
		DescriptionEN:   "Counts spec layouts, blocked specs, legacy and unknown review policy keys, sealed bundles, attestations by identity mode and attribution, and incomplete transfers, with the action and removal criterion of each class. Writes nothing; 6.x has no apply.",
		DescriptionPtBR: "Conta layouts de spec, specs blocked, chaves legadas e desconhecidas da política de review, bundles selados, atestações por modo de identidade e atribuição, e transferências incompletas, com a ação e o critério de remoção de cada classe. Não escreve nada; a 6.x não tem apply.",
		Flags: []FlagHelp{
			{"--dry-run", "Required: inventory only", "Obrigatório: apenas inventário"},
			{"--json", "Output the inventory as JSON", "Emite o inventário em JSON"},
		},
		Examples: []string{
			"pose migrate v7 --dry-run",
		},
	},
	"stats": {
		Name:            "stats",
		SummaryEN:       "Display historical POSE engineering statistics and task metrics",
		SummaryPtBR:     "Exibe estatísticas históricas de engenharia e métricas de tarefas do POSE",
		Usage:           "pose stats replay [--limit N] [--json] | pose stats [workflows|tasks|contexts] [--since-days N] [--json] | pose stats governance [--since-days N] [--maturity-days N] [--min-sample N] [--waits] [--rework] [--outcomes] [--json]",
		DescriptionEN:   "Aggregates historical outcomes, or (with governance) reports separate preparation, judgment, intervention, freshness and coverage dimensions without a quality score.",
		DescriptionPtBR: "Agrega resultados históricos ou, com governance, separa preparação, julgamento, intervenção, atualidade e cobertura sem score de qualidade.",
		Flags: []FlagHelp{
			{"--since-days <N>", "Analyze historical data within the last N days (default: 30)", "Analisa dados históricos dos últimos N dias (padrão: 30)"},
			{"--maturity-days <N>", "Governance query maturity window (default: 30)", "Janela de maturidade da consulta governance (padrão: 30)"},
			{"--min-sample <N>", "Governance query minimum sample (default: 3)", "Amostra mínima da consulta governance (padrão: 3)"},
			{"--waits", "Governance: request age, attributed wait and known blocking, kept apart", "Governance: idade das solicitações, espera atribuída e bloqueio conhecido, separados"},
			{"--rework", "Governance: why review work was redone, by what changed; unknown otherwise", "Governance: por que o trabalho de review foi refeito, pelo que mudou; senão desconhecido"},
			{"--outcomes", "Governance: with --waits/--rework, also compute the outcomes (reads the whole history)", "Governance: com --waits/--rework, calcula também os resultados (lê todo o histórico)"},
			{"--json", "Output statistics in JSON format", "Emite as estatísticas em formato JSON"},
		},
		Examples: []string{
			"pose stats",
			"pose stats tasks --since-days 14",
			"pose stats governance --json",
			"pose stats replay --json",
		},
	},
	"usage": {
		Name:            "usage",
		SummaryEN:       "Inspect local CLI and MCP tool usage telemetry and outcome metrics",
		SummaryPtBR:     "Inspeciona métricas locais de uso e sucesso de comandos CLI e MCP",
		Usage:           "pose usage [--since-days N] [--tool <name>] [--surface cli|mcp] [--json] | pose usage adjudicate --tool NAME --finding ID --verdict valid|wont-fix|false-positive --reason TEXT --by ALIAS",
		DescriptionEN:   "Reports local tool invocation counts, error rates, average latency, and structured finding lifecycle without external network reporting.",
		DescriptionPtBR: "Informa contagens de invocação de ferramentas, taxas de erro, latência média e ciclo de achados sem envio externo de dados.",
		Flags: []FlagHelp{
			{"--since-days <N>", "Filter usage data to the last N days", "Filtra os dados de uso para os últimos N dias"},
			{"--tool <name>", "Filter report to a specific CLI command or MCP tool name", "Filtra o relatório para uma ferramenta ou comando específico"},
			{"--surface <cli|mcp>", "Filter by invocation interface surface", "Filtra pela interface de invocação (cli ou mcp)"},
			{"--json", "Output usage telemetry in JSON format", "Emite a telemetria de uso em formato JSON"},
			{"adjudicate", "Append a human finding verdict to the project journal", "Registra um veredito humano no diário do projeto"},
		},
		Examples: []string{
			"pose usage --surface mcp",
			"pose usage --tool validate --since-days 7",
			"pose usage adjudicate --tool validate --finding check-a --verdict false-positive --reason 'Reviewed result' --by reviewer",
		},
	},
	"dora-metrics": {
		Name:            "dora-metrics",
		SummaryEN:       "Calculate DORA delivery metrics from recorded deployments and incidents",
		SummaryPtBR:     "Calcula métricas DORA a partir de deploys e incidentes registrados",
		Usage:           "pose dora-metrics [--application <name>] [--environment <env>] [--window-days N] [--json]",
		DescriptionEN:   "Computes the 5 DORA metrics (Deployment Frequency, Lead Time for Changes, Change Failure Rate, Time to Restore Service, Reliability) from local governed events.",
		DescriptionPtBR: "Calcula as 5 métricas DORA (Frequência de Deploy, Tempo de Lead Time, Taxa de Falha de Mudança, Tempo de Recuperação, Confiabilidade) a partir de eventos locais.",
		Flags: []FlagHelp{
			{"--application <name>", "Scope calculation to a specific application", "Restringe o cálculo a uma aplicação específica"},
			{"--environment <env>", "Environment to measure (default: production)", "Ambiente medido (padrão: production)"},
			{"--window-days <N>", "Calculate metrics over the trailing N days (default: 30)", "Calcula as métricas sobre os últimos N dias (padrão: 30)"},
			{"--json", "Output DORA metrics in JSON format", "Emite as métricas DORA em formato JSON"},
		},
		Examples: []string{
			"pose dora-metrics --environment production",
		},
	},
	"adoption-metrics": {
		Name:            "adoption-metrics",
		SummaryEN:       "Derive POSE framework adoption and retention indicators",
		SummaryPtBR:     "Deriva indicadores de adoção e retenção do framework POSE",
		Usage:           "pose adoption-metrics [--json]",
		DescriptionEN:   "Computes framework activation, gate utilization rate, time-to-first-gate, and task velocity across active engineering workspaces.",
		DescriptionPtBR: "Calcula ativação do framework, taxa de utilização de gates, tempo até o primeiro gate e velocidade de entrega.",
		Flags: []FlagHelp{
			{"--json", "Output adoption indicators in JSON format", "Emite os indicadores de adoção em formato JSON"},
		},
		Examples: []string{
			"pose adoption-metrics",
		},
	},
	"history-check": {
		Name:            "history-check",
		SummaryEN:       "Verify that all history log files are properly tracked in Git",
		SummaryPtBR:     "Verifica se todos os arquivos de log histórico estão rastreados no Git",
		Usage:           "pose history-check [--strict|--tolerant] [--json] [--json-out <path>] [--quiet] [--color auto|always|never]",
		DescriptionEN:   "Ensures that all append-only task history files under .pose/reports/history/ are committed and tracked in version control.",
		DescriptionPtBR: "Garante que todos os arquivos históricos de tarefas sob .pose/reports/history/ estejam commitados e rastreados no controle de versão.",
		Flags: []FlagHelp{
			{"--json", "Print one JSON document with the verdict, findings and counts instead of the human report", "Imprime um documento JSON com veredito, findings e contagens no lugar do relatório humano"},
			{"--json-out <path>", "Also write the JSON document to a project-relative file, keeping the human report", "Também grava o documento JSON num arquivo relativo ao projeto, mantendo o relatório humano"},
			{"--quiet", "Print the verdict alone", "Imprime apenas o veredito"},
			{"--color auto|always|never", "Force or suppress colour; NO_COLOR and POSE_COLOR are honoured", "Força ou suprime cor; NO_COLOR e POSE_COLOR são respeitados"},
		},
		Examples: []string{
			"pose history-check",
		},
	},
	"knowledge-check": {
		Name:            "knowledge-check",
		SummaryEN:       "Check knowledge schema validity, TTL expiration, and overdue reviews",
		SummaryPtBR:     "Verifica validade de schema, expiração de TTL e revisões vencidas de conhecimento",
		Usage:           "pose knowledge-check [--strict|--tolerant] [--max-overdue N] [--json] [--json-out <path>] [--quiet] [--color auto|always|never]",
		DescriptionEN:   "Enforces schema validity and TTL limits across handoffs, decision logs, and notes under .pose/knowledge/, failing when unreviewed knowledge exceeds the overdue threshold.",
		DescriptionPtBR: "Valida o schema e limites de TTL de handoffs, decision logs e notes sob .pose/knowledge/, falhando quando itens vencidos ultrapassam o limite.",
		Flags: []FlagHelp{
			{"--max-overdue <N>", "Maximum allowed number of overdue knowledge items before failing (default: 0)", "Número máximo de itens vencidos tolerados antes de falhar (padrão: 0)"},
			{"--strict", "Fail on any expired TTL or missing owner metadata", "Falha em qualquer TTL expirado ou metadados de responsável ausentes"},
			{"--json", "Print one JSON document with the verdict, findings and counts instead of the human report", "Imprime um documento JSON com veredito, findings e contagens no lugar do relatório humano"},
			{"--json-out <path>", "Also write the JSON document to a project-relative file, keeping the human report", "Também grava o documento JSON num arquivo relativo ao projeto, mantendo o relatório humano"},
			{"--quiet", "Print the verdict alone", "Imprime apenas o veredito"},
			{"--color auto|always|never", "Force or suppress colour; NO_COLOR and POSE_COLOR are honoured", "Força ou suprime cor; NO_COLOR e POSE_COLOR são respeitados"},
		},
		Examples: []string{
			"pose knowledge-check",
			"pose knowledge-check --strict",
		},
	},
	"recurrence-check": {
		Name:            "recurrence-check",
		SummaryEN:       "Detect recurring task failures and trigger systemic escalation",
		SummaryPtBR:     "Detecta falhas recorrentes de tarefas e dispara escalação sistêmica",
		Usage:           "pose recurrence-check [--strict|--tolerant] [--window-days N] [--threshold T] [--include-pass] [--json] [--json-out <path>] [--quiet] [--color auto|always|never]",
		DescriptionEN:   "Scans execution logs to identify task slugs that fail repeatedly, recommending systemic escalation into new domain rules or workflows.",
		DescriptionPtBR: "Escaneia logs de execução para identificar tarefas que falham repetidamente, recomendando escalação sistêmica em novas regras ou workflows.",
		Flags: []FlagHelp{
			{"--window-days <N>", "Time window in days to evaluate recurring failures (default: 14)", "Janela de tempo em dias para avaliar falhas recorrentes (padrão: 14)"},
			{"--threshold <T>", "Number of occurrences required to trigger an escalation notice (default: 3)", "Número de ocorrências necessárias para disparar aviso de escalação (padrão: 3)"},
			{"--json", "Print one JSON document with the verdict, findings and counts instead of the human report", "Imprime um documento JSON com veredito, findings e contagens no lugar do relatório humano"},
			{"--json-out <path>", "Also write the JSON document to a project-relative file, keeping the human report", "Também grava o documento JSON num arquivo relativo ao projeto, mantendo o relatório humano"},
			{"--quiet", "Print the verdict alone", "Imprime apenas o veredito"},
			{"--color auto|always|never", "Force or suppress colour; NO_COLOR and POSE_COLOR are honoured", "Força ou suprime cor; NO_COLOR e POSE_COLOR são respeitados"},
		},
		Examples: []string{
			"pose recurrence-check",
		},
	},
	"skills-check": {
		Name:            "skills-check",
		SummaryEN:       "Validate agent skill definitions, symlinks, and frontmatter parity",
		SummaryPtBR:     "Valida definições de skills de agente, symlinks e paridade de frontmatter",
		Usage:           "pose skills-check [--strict|--tolerant] [--json] [--json-out <path>] [--quiet] [--color auto|always|never]",
		DescriptionEN:   "Verifies that all skills in .agents/skills/ have valid SKILL.md frontmatter, required reading sections, and synchronized symlinks in .claude/skills/.",
		DescriptionPtBR: "Verifica se todas as skills em .agents/skills/ possuem frontmatter válido no SKILL.md, seções obrigatórias de leitura e symlinks sincronizados em .claude/skills/.",
		Flags: []FlagHelp{
			{"--json", "Print one JSON document with the verdict, findings and counts instead of the human report", "Imprime um documento JSON com veredito, findings e contagens no lugar do relatório humano"},
			{"--json-out <path>", "Also write the JSON document to a project-relative file, keeping the human report", "Também grava o documento JSON num arquivo relativo ao projeto, mantendo o relatório humano"},
			{"--quiet", "Print the verdict alone", "Imprime apenas o veredito"},
			{"--color auto|always|never", "Force or suppress colour; NO_COLOR and POSE_COLOR are honoured", "Força ou suprime cor; NO_COLOR e POSE_COLOR são respeitados"},
		},
		Examples: []string{
			"pose skills-check",
		},
	},
	"report": {
		Name:            "report",
		SummaryEN:       "Record an engineering execution report and append-only history entry",
		SummaryPtBR:     "Grava um relatório de execução de engenharia e entrada no histórico",
		Usage:           "pose report --task \"<title>\" [--outcome pass|fail|partial|skipped|unknown] [--since <ref>] [--git-stage] [options]",
		DescriptionEN:   "Generates a structured execution report and appends an immutable JSONL record into .pose/reports/history/ capturing changes and validation evidence.",
		DescriptionPtBR: "Gera um relatório estruturado de execução e anexa um registro imutável em JSONL sob .pose/reports/history/ com alterações e evidências.",
		Flags: []FlagHelp{
			{"--task <title>", "Descriptive summary of the executed engineering task", "Resumo descritivo da tarefa de engenharia executada"},
			{"--outcome <pass|fail|partial|skipped|unknown>", "Execution outcome (default: derived from the validation log, otherwise unknown)", "Resultado da execução (padrão: derivado do log de validação, senão unknown)"},
			{"--spec <slug>", "Associate the report with a specific specification", "Associa o relatório a uma especificação específica"},
			{"--git-stage", "Automatically stage (git add) the generated history JSONL file", "Executa git add automaticamente no arquivo JSONL de histórico gerado"},
			{"--type <standard|doc-audit>", "Report type, which also names the report and history files (default: standard)", "Tipo do relatório, que também nomeia os arquivos de relatório e histórico (padrão: standard)"},
			{"--validate-output <path>", "Validation log to read commands, results and the derived outcome from; must stay inside the project (default: .pose/reports/pose-validate.latest.log, then .pose/pose-validate.log)", "Log de validação de onde ler comandos, resultados e o resultado derivado; deve ficar dentro do projeto (padrão: .pose/reports/pose-validate.latest.log, depois .pose/pose-validate.log)"},
			{"--since <ref>", "List the files changed since this Git ref (default: uncommitted changes)", "Lista os arquivos alterados desde esta ref Git (padrão: alterações não commitadas)"},
			{"--change-from <ref>", "Start of the commit range recorded as the spec's immutable change set; requires --spec", "Início do intervalo de commits registrado como change set imutável da spec; exige --spec"},
			{"--change-to <ref>", "End of that commit range; requires --spec", "Fim desse intervalo de commits; exige --spec"},
			{"--workflow <name>", "Workflow followed, recorded in the report", "Workflow seguido, registrado no relatório"},
			{"--rules <a,b>", "Comma-separated rules applied, recorded in the report", "Regras aplicadas, separadas por vírgula, registradas no relatório"},
			{"--risk <text>", "Risks, one per line, recorded in the report", "Riscos, um por linha, registrados no relatório"},
			{"--context <text>", "Execution context recorded in the report metadata (default: not-provided)", "Contexto de execução registrado nos metadados do relatório (padrão: not-provided)"},
			{"--validation-profile <name>", "Validation profile recorded in the report metadata (default: not-provided)", "Perfil de validação registrado nos metadados do relatório (padrão: not-provided)"},
			{"--duration-seconds <n>", "Optional telemetry: how long the task took, read by recurrence-effect", "Telemetria opcional: quanto a tarefa levou, lida por recurrence-effect"},
			{"--cost-usd <n>", "Optional telemetry: what the task cost in USD, read by recurrence-effect", "Telemetria opcional: quanto a tarefa custou em USD, lida por recurrence-effect"},
		},
		Examples: []string{
			"pose report --task \"Implement token caching\" --outcome pass --git-stage",
		},
	},
}
