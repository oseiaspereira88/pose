package cli

// FlagHelp describes a single command flag or option.
type FlagHelp struct {
	Flag            string
	DescriptionEN   string
	DescriptionPtBR string
}

// SubcommandHelp describes a subcommand within a command group.
type SubcommandHelp struct {
	Name            string
	Usage           string
	SummaryEN       string
	SummaryPtBR     string
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

// commandHelpCatalog maps command names to their structured help definitions.
var commandHelpCatalog = map[string]CommandHelp{
	"init": {
		Name:            "init",
		SummaryEN:       "Initialize POSE structure in the current repository",
		SummaryPtBR:     "Inicializa a estrutura do POSE no repositório atual",
		Usage:           "pose init [--wizard [--yes]]",
		DescriptionEN:   "Ensures the minimum required .pose directory structure, policy files, and indexes. When --wizard is provided, scans the repository to auto-detect stacks and seed the validation matrix.",
		DescriptionPtBR: "Garante a estrutura mínima necessária de diretórios, políticas e índices do .pose. Quando --wizard é informado, escaneia o repositório para auto-detectar stacks e semear a matriz de validação.",
		Flags: []FlagHelp{
			{"--wizard", "Run interactive onboarding wizard to detect modules and stack rules", "Executa o assistente de onboarding para detectar módulos e regras de stack"},
			{"--yes", "Auto-accept wizard prompts with recommended defaults", "Aceita automaticamente as perguntas do assistente com os padrões recomendados"},
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
		Usage:           "pose validate [--strict|--tolerant] [--stack <s>] [--module <p>] [--report] [--json <path>]",
		DescriptionEN:   "Executes deterministic verification commands (tests, linters, typechecks, builds) declared in .pose/indexes/validation-matrix.json.",
		DescriptionPtBR: "Executa comandos de verificação determinísticos (testes, linters, checagens de tipo, builds) declarados em .pose/indexes/validation-matrix.json.",
		Flags: []FlagHelp{
			{"--strict", "Fail on any error or structural validation warning (default in CI)", "Falha em qualquer erro ou aviso estrutural de validação (padrão em CI)"},
			{"--tolerant", "Allow non-blocking warnings while failing on required errors", "Permite avisos não-bloqueantes, falhando apenas em erros obrigatórios"},
			{"--stack <name>", "Filter execution to modules matching the given stack (e.g. go, node, python)", "Filtra a execução para módulos da stack informada"},
			{"--module <path>", "Filter execution to a specific module directory path", "Filtra a execução para o caminho de um módulo específico"},
			{"--report", "Persist validation findings into .pose/reports/", "Persiste os achados de validação sob .pose/reports/"},
			{"--json <path>", "Write structured validation outcome to the specified JSON path", "Grava o resultado da validação no caminho JSON especificado"},
			{"--changed-from <rev>", "Validate only modules affected between git revisions", "Valida apenas módulos afetados entre as revisões git"},
		},
		Examples: []string{
			"pose validate --strict",
			"pose validate --module pose-mcp --strict",
			"pose validate --json .pose/results/delivery-validation.json",
		},
	},
	"check": {
		Name:            "check",
		SummaryEN:       "Verify structural integrity, matrix schema, task maps, and specs",
		SummaryPtBR:     "Verifica integridade estrutural, schema da matriz, task maps e specs",
		Usage:           "pose check [--strict|--tolerant]",
		DescriptionEN:   "Performs comprehensive structural verification on all POSE files, broken markdown links, frontmatter syntax, matrix JSON schemas, and spec graphs.",
		DescriptionPtBR: "Realiza verificação estrutural completa em todos os arquivos do POSE, links quebrados em markdown, sintaxe de frontmatter, schemas JSON e grafos de specs.",
		Flags: []FlagHelp{
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
		Usage:           "pose lint-spec <slug>|--all [--strict|--tolerant] [--ready-check] [--required-only]",
		DescriptionEN:   "Validates that a spec document conforms to the 7-section template, has stable requirement IDs (R1, R2), complete frontmatter, and valid traceability evidence upon closeout.",
		DescriptionPtBR: "Valida se a spec está em conformidade com o template de 7 seções, IDs estáveis de requisitos (R1, R2), frontmatter completo e evidências válidas de rastreabilidade.",
		Flags: []FlagHelp{
			{"--ready-check", "Enforce Definition of Ready (DoR) gate before transitioning to in-progress", "Aplica o gate de Definition of Ready (DoR) antes de transicionar para in-progress"},
			{"--strict", "Fail on any missing requirement trace or unfilled section in done specs", "Falha em qualquer rastreio ausente ou seção não preenchida em specs concluídas"},
			{"--all", "Lint all specifications present under .pose/specs/", "Valida todas as especificações presentes sob .pose/specs/"},
			{"--required-only", "Check only mandatory core sections without optional decisions", "Verifica apenas seções obrigatórias sem decisões opcionais"},
		},
		Examples: []string{
			"pose lint-spec my-feature --ready-check",
			"pose lint-spec my-feature --strict",
			"pose lint-spec --all --strict",
		},
	},
	"new-spec": {
		Name:            "new-spec",
		SummaryEN:       "Scaffold a new feature specification",
		SummaryPtBR:     "Cria o scaffold de uma nova especificação de feature",
		Usage:           "pose new-spec <slug>",
		DescriptionEN:   "Generates a new draft spec scaffold under .pose/specs/<slug>/spec.md with standard lifecycle frontmatter and the 7 required engineering sections.",
		DescriptionPtBR: "Gera o scaffold de uma nova spec em rascunho sob .pose/specs/<slug>/spec.md com frontmatter padrão e as 7 seções de engenharia.",
		Examples: []string{
			"pose new-spec user-authentication",
			"pose new-spec billing-export-pipeline",
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
		Usage:           "pose review <bundle|auto-attest|attest|verify|record> <scope> [options]",
		DescriptionEN:   "Manages the component-aware review process, sealing immutable review bundles (rvb-*) and separate attestations (rva-*) before spec closeout.",
		DescriptionPtBR: "Gerencia o processo de review por componentes, selando bundles imutáveis (rvb-*) e atestações separadas (rva-*) antes do fechamento da spec.",
		Subcommands: []SubcommandHelp{
			{"bundle", "pose review bundle <scope> [--seal] [--explain]", "Prepare or seal an immutable review subject bundle", "Prepara ou sela o bundle imutável de revisão"},
			{"auto-attest", "pose review auto-attest <scope|bundle-id> [--apply]", "Extract validation results and automatically record attestation", "Extrai resultados de validação e registra atestação automaticamente"},
			{"attest", "pose review attest <bundle-id> --reviewer <id> --decision <decision> --evidence <ref> [--apply]", "Record a manual review decision attestation", "Registra manualmente uma atestação de decisão de review"},
			{"verify", "pose review verify <scope|bundle-id>", "Verify freshness and closeout readiness of review bundles", "Verifica atualidade e prontidão do review bundle para fechamento"},
			{"record", "pose review record <scope> ...", "Compatibility entrypoint for review recording", "Ponto de entrada de compatibilidade para registro de review"},
		},
		Examples: []string{
			"pose review bundle spec:my-feature --seal",
			"pose review auto-attest spec:my-feature --apply",
			"pose review verify spec:my-feature",
		},
	},
	"close": {
		Name:            "close",
		SummaryEN:       "Apply review-gated lifecycle closeout to a spec or milestone",
		SummaryPtBR:     "Aplica fechamento governado com portão de review em spec ou marco",
		Usage:           "pose close <scope> [--json]",
		DescriptionEN:   "Transitions a completed specification (or milestone) to status: done after verifying that review attestations, requirement traces, and delivery assurances are fully satisfied.",
		DescriptionPtBR: "Transiciona uma spec (ou marco) para status: done após verificar se atestações de review, rastreio de requisitos e garantias de entrega foram cumpridos.",
		Flags: []FlagHelp{
			{"--json", "Output closeout transition details in JSON format", "Emite os detalhes da transição de fechamento em formato JSON"},
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
		DescriptionEN:   "Controls POSE Contributor Mode, signaling executing AI agents to automatically stage sanitized feedback artifacts under .pose/contributions/ without leaking proprietary code.",
		DescriptionPtBR: "Controla o Modo Contribuidor do POSE, sinalizando agentes de IA para registrar rascunhos de feedback sob .pose/contributions/ sem vazar código privado.",
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
			{"record", "pose release record --version vX.Y.Z --provider-state <state>", "Import and record provider release publication evidence", "Registra evidência de publicação da release no provedor"},
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
		Usage:           "pose state [init|refresh|diff] [--if-stale] [--json]",
		DescriptionEN:   "Generates, updates, and diffs the native .pose/state/project-state.json artifact capturing specs, debts, and repository health.",
		DescriptionPtBR: "Gera, atualiza e compara o artefato nativo .pose/state/project-state.json com specs, débitos e integridade do repositório.",
		Flags: []FlagHelp{
			{"--if-stale", "Refresh state only if the existing artifact exceeds the staleness threshold", "Atualiza o estado apenas se o artefato existente ultrapassar o limite de desatualização"},
			{"--json", "Output state representation in JSON format", "Emite a representação do estado em formato JSON"},
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
		Usage:           "pose followups [--open|--all] [--json] [--overdue]",
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
		Usage:           "pose index",
		DescriptionEN:   "Recomputes and writes all static indexes including spec-graph.json, roadmaps.json, releases.json, and delivery-integrity.json.",
		DescriptionPtBR: "Recalcula e grava todos os índices estáticos incluindo spec-graph.json, roadmaps.json, releases.json e delivery-integrity.json.",
		Examples: []string{
			"pose index",
		},
	},
	"update": {
		Name:            "update",
		SummaryEN:       "Update instance contracts and machinery to match engine version",
		SummaryPtBR:     "Atualiza contratos e maquinário da instância para a versão do motor",
		Usage:           "pose update [--dry-run] [--force] [--schema-only]",
		DescriptionEN:   "Applies idempotent schema migrations and merges canonical workflow, rule, and manual updates while preserving instance-owned and contributor sections.",
		DescriptionPtBR: "Aplica migrações idempotentes de schema e mescla atualizações de manuais, workflows e regras, preservando seções da instância e do modo contribuidor.",
		Flags: []FlagHelp{
			{"--dry-run", "Preview file merges and migrations without writing changes", "Visualiza as mesclagens e migrações sem gravar alterações"},
			{"--force", "Overwrite managed manuals wholesale instead of performing three-way merge", "Sobrescreve os manuais por completo sem realizar mesclagem"},
			{"--schema-only", "Bump schema version stamp without updating docs and workflows", "Atualiza apenas o número de versão de schema sem alterar docs"},
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
		Usage:           "pose assess <discover|integrate|tech-debt> [--json] [--update-state]",
		DescriptionEN:   "Runs deep static scanners to assess component LOC, dependencies, technical debt markers (TODO, FIXME, panic), and public contract interfaces.",
		DescriptionPtBR: "Executa scanners estáticos profundos para avaliar LOC dos componentes, dependências, marcadores de débito técnico (TODO, FIXME, panic) e contratos públicos.",
		Subcommands: []SubcommandHelp{
			{"discover", "pose assess discover [--component <dir>] [--update-state]", "Discover LOC metrics, debts, and module structure", "Descobre métricas de LOC, débitos e estrutura de módulos"},
			{"integrate", "pose assess integrate", "Check inter-module contracts (REST, Protobuf, Kafka, MCP)", "Verifica contratos inter-módulos (REST, Protobuf, Kafka, MCP)"},
			{"tech-debt", "pose assess tech-debt", "Scan codebase for technical debt markers and uncovered stubs", "Escaneia a base de código por marcadores de débito e stubs"},
		},
		Examples: []string{
			"pose assess discover --update-state",
			"pose assess tech-debt",
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
	"stats": {
		Name:            "stats",
		SummaryEN:       "Display historical POSE engineering statistics and task metrics",
		SummaryPtBR:     "Exibe estatísticas históricas de engenharia e métricas de tarefas do POSE",
		Usage:           "pose stats [workflows|tasks|contexts] [--since-days N] [--json]",
		DescriptionEN:   "Aggregates execution metrics from .pose/reports/history/, calculating task completion rates, pass/fail ratios, and execution frequency.",
		DescriptionPtBR: "Agrega métricas de execução a partir de .pose/reports/history/, calculando taxas de conclusão de tarefas, taxa de sucesso e frequência.",
		Flags: []FlagHelp{
			{"--since-days <N>", "Analyze historical data within the last N days (default: 30)", "Analisa dados históricos dos últimos N dias (padrão: 30)"},
			{"--json", "Output statistics in JSON format", "Emite as estatísticas em formato JSON"},
		},
		Examples: []string{
			"pose stats",
			"pose stats tasks --since-days 14",
		},
	},
	"usage": {
		Name:            "usage",
		SummaryEN:       "Inspect local CLI and MCP tool usage telemetry and outcome metrics",
		SummaryPtBR:     "Inspeciona métricas locais de uso e sucesso de comandos CLI e MCP",
		Usage:           "pose usage [--since-days N] [--tool <name>] [--surface cli|mcp] [--json]",
		DescriptionEN:   "Reports local tool invocation counts, error rates, average latency, and structured finding lifecycle without external network reporting.",
		DescriptionPtBR: "Informa contagens de invocação de ferramentas, taxas de erro, latência média e ciclo de achados sem envio externo de dados.",
		Flags: []FlagHelp{
			{"--since-days <N>", "Filter usage data to the last N days", "Filtra os dados de uso para os últimos N dias"},
			{"--tool <name>", "Filter report to a specific CLI command or MCP tool name", "Filtra o relatório para uma ferramenta ou comando específico"},
			{"--surface <cli|mcp>", "Filter by invocation interface surface", "Filtra pela interface de invocação (cli ou mcp)"},
			{"--json", "Output usage telemetry in JSON format", "Emite a telemetria de uso em formato JSON"},
		},
		Examples: []string{
			"pose usage --surface mcp",
			"pose usage --tool validate --since-days 7",
		},
	},
	"dora-metrics": {
		Name:            "dora-metrics",
		SummaryEN:       "Calculate DORA delivery metrics from recorded deployments and incidents",
		SummaryPtBR:     "Calcula métricas DORA a partir de deploys e incidentes registrados",
		Usage:           "pose dora-metrics [--app <name>] [--env <env>] [--since-days N] [--json]",
		DescriptionEN:   "Computes the 5 DORA metrics (Deployment Frequency, Lead Time for Changes, Change Failure Rate, Time to Restore Service, Reliability) from local governed events.",
		DescriptionPtBR: "Calcula as 5 métricas DORA (Frequência de Deploy, Tempo de Lead Time, Taxa de Falha de Mudança, Tempo de Recuperação, Confiabilidade) a partir de eventos locais.",
		Flags: []FlagHelp{
			{"--app <name>", "Scope calculation to a specific application ID", "Restringe o cálculo a uma aplicação específica"},
			{"--env <name>", "Filter by environment (e.g. production, staging)", "Filtra pelo ambiente especificado"},
			{"--since-days <N>", "Calculate metrics over the trailing N days (default: 90)", "Calcula as métricas sobre a janela dos últimos N dias (padrão: 90)"},
			{"--json", "Output DORA metrics in JSON format", "Emite as métricas DORA em formato JSON"},
		},
		Examples: []string{
			"pose dora-metrics --env production",
		},
	},
	"adoption-metrics": {
		Name:            "adoption-metrics",
		SummaryEN:       "Derive POSE framework adoption and retention indicators",
		SummaryPtBR:     "Deriva indicadores de adoção e retenção do framework POSE",
		Usage:           "pose adoption-metrics [--since-days N] [--json]",
		DescriptionEN:   "Computes framework activation, gate utilization rate, time-to-first-gate, and task velocity across active engineering workspaces.",
		DescriptionPtBR: "Calcula ativação do framework, taxa de utilização de gates, tempo até o primeiro gate e velocidade de entrega.",
		Flags: []FlagHelp{
			{"--since-days <N>", "Analyze adoption data over the last N days (default: 90)", "Analisa dados de adoção nos últimos N dias (padrão: 90)"},
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
		Usage:           "pose history-check [--strict|--tolerant]",
		DescriptionEN:   "Ensures that all append-only task history files under .pose/reports/history/ are committed and tracked in version control.",
		DescriptionPtBR: "Garante que todos os arquivos históricos de tarefas sob .pose/reports/history/ estejam commitados e rastreados no controle de versão.",
		Examples: []string{
			"pose history-check",
		},
	},
	"knowledge-check": {
		Name:            "knowledge-check",
		SummaryEN:       "Check knowledge schema validity, TTL expiration, and overdue reviews",
		SummaryPtBR:     "Verifica validade de schema, expiração de TTL e revisões vencidas de conhecimento",
		Usage:           "pose knowledge-check [--strict|--tolerant] [--max-overdue N]",
		DescriptionEN:   "Enforces schema validity and TTL limits across handoffs, decision logs, and notes under .pose/knowledge/, failing when unreviewed knowledge exceeds the overdue threshold.",
		DescriptionPtBR: "Valida o schema e limites de TTL de handoffs, decision logs e notes sob .pose/knowledge/, falhando quando itens vencidos ultrapassam o limite.",
		Flags: []FlagHelp{
			{"--max-overdue <N>", "Maximum allowed number of overdue knowledge items before failing (default: 0)", "Número máximo de itens vencidos tolerados antes de falhar (padrão: 0)"},
			{"--strict", "Fail on any expired TTL or missing owner metadata", "Falha em qualquer TTL expirado ou metadados de responsável ausentes"},
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
		Usage:           "pose recurrence-check [--strict|--tolerant] [--window-days N] [--threshold T] [--include-pass]",
		DescriptionEN:   "Scans execution logs to identify task slugs that fail repeatedly, recommending systemic escalation into new domain rules or workflows.",
		DescriptionPtBR: "Escaneia logs de execução para identificar tarefas que falham repetidamente, recomendando escalação sistêmica em novas regras ou workflows.",
		Flags: []FlagHelp{
			{"--window-days <N>", "Time window in days to evaluate recurring failures (default: 14)", "Janela de tempo em dias para avaliar falhas recorrentes (padrão: 14)"},
			{"--threshold <T>", "Number of occurrences required to trigger an escalation notice (default: 3)", "Número de ocorrências necessárias para disparar aviso de escalação (padrão: 3)"},
		},
		Examples: []string{
			"pose recurrence-check",
		},
	},
	"skills-check": {
		Name:            "skills-check",
		SummaryEN:       "Validate agent skill definitions, symlinks, and frontmatter parity",
		SummaryPtBR:     "Valida definições de skills de agente, symlinks e paridade de frontmatter",
		Usage:           "pose skills-check [--strict|--tolerant]",
		DescriptionEN:   "Verifies that all skills in .agents/skills/ have valid SKILL.md frontmatter, required reading sections, and synchronized symlinks in .claude/skills/.",
		DescriptionPtBR: "Verifica se todas as skills em .agents/skills/ possuem frontmatter válido no SKILL.md, seções obrigatórias de leitura e symlinks sincronizados em .claude/skills/.",
		Examples: []string{
			"pose skills-check",
		},
	},
	"report": {
		Name:            "report",
		SummaryEN:       "Record an engineering execution report and append-only history entry",
		SummaryPtBR:     "Grava um relatório de execução de engenharia e entrada no histórico",
		Usage:           "pose report --task \"<title>\" [--outcome pass|fail|partial] [--since <ref>] [--git-stage] [options]",
		DescriptionEN:   "Generates a structured execution report and appends an immutable JSONL record into .pose/reports/history/ capturing changes and validation evidence.",
		DescriptionPtBR: "Gera um relatório estruturado de execução e anexa um registro imutável em JSONL sob .pose/reports/history/ com alterações e evidências.",
		Flags: []FlagHelp{
			{"--task <title>", "Descriptive summary of the executed engineering task", "Resumo descritivo da tarefa de engenharia executada"},
			{"--outcome <pass|fail|partial>", "Execution outcome status (default: auto-derived from validation)", "Status de resultado da execução (padrão: auto-derivado da validação)"},
			{"--git-stage", "Automatically stage (git add) the generated history JSONL file", "Executa git add automaticamente no arquivo JSONL de histórico gerado"},
			{"--spec <slug>", "Associate the report with a specific specification", "Associa o relatório a uma especificação específica"},
		},
		Examples: []string{
			"pose report --task \"Implement token caching\" --outcome pass --git-stage",
		},
	},
}
