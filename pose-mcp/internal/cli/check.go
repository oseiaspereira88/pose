package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/harne8/pose-mcp/internal/pose"
)

const nativeSchemaVersion = 1

var checkReference = regexp.MustCompile(`(?:^|[\s\x60"'(<])(\.pose/[A-Za-z0-9_./-]*|\.agents/skills/[A-Za-z0-9_./-]*|local/[A-Za-z0-9_./-]*)`)
var checkSlug = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
var checkMilestoneRef = regexp.MustCompile(`^milestone:[a-z0-9][a-z0-9._-]*/[a-z0-9][a-z0-9._-]*$`)
var checkRoadmapRef = regexp.MustCompile(`^roadmap:[a-z0-9][a-z0-9._-]*$`)

type nativeChecker struct {
	root     string
	mode     string
	locale   cliLocale
	stdout   io.Writer
	errors   int
	warnings int
}

func (checker *nativeChecker) message(english, portuguese string) string {
	return cliText(checker.locale, english, portuguese)
}

func (checker *nativeChecker) issue(level, message string) {
	fmt.Fprintf(checker.stdout, "[%s] %s\n", level, message)
	if level == "ERRO" {
		checker.errors++
	} else {
		checker.warnings++
	}
}

func (checker *nativeChecker) failOrWarn(message string) {
	if checker.mode == "tolerant" {
		checker.issue("AVISO", message)
	} else {
		checker.issue("ERRO", message)
	}
}

func cmdCheck(root string, args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	mode := "strict"
	if len(args) > 1 {
		fmt.Fprintln(stderr, cliText(locale, "Usage: pose check [--strict|--tolerant]", "Uso: pose check [--strict|--tolerant]"))
		return 2
	}
	if len(args) == 1 {
		switch args[0] {
		case "--strict":
		case "--tolerant":
			mode = "tolerant"
		default:
			fmt.Fprintf(stderr, cliText(locale, "Error: invalid argument: %s\n", "Erro: argumento inválido: %s\n"), args[0])
			return 2
		}
	}
	checker := &nativeChecker{root: root, mode: mode, locale: locale, stdout: stdout}
	checker.checkRequiredStructure()
	checker.checkSchemaVersion()
	checker.checkReferences()
	checker.checkValidationMatrix()
	checker.checkTaskMap()
	checker.checkSpecs()
	checker.checkChangelogs()
	checker.checkReadyTransitions()
	checker.checkCapabilities()
	if checker.errors > 0 {
		fmt.Fprintf(stdout, "Resultado: FALHA — estrutura POSE com %d erro(s).\n", checker.errors)
		return 1
	}
	if checker.warnings > 0 {
		fmt.Fprintf(stdout, "Resultado: SUCESSO (modo tolerant) com %d aviso(s).\n", checker.warnings)
		return 0
	}
	fmt.Fprintf(stdout, "Resultado: SUCESSO — estrutura POSE válida (modo %s).\n", mode)
	return 0
}

func (checker *nativeChecker) checkRequiredStructure() {
	required := []string{"AGENTS.md", "POSE.md", ".pose", ".pose/workflows", ".pose/templates", ".pose/rules", ".pose/workflows/feature.md", ".pose/workflows/review.md", ".pose/workflows/bugfix.md", ".pose/templates/spec.md"}
	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(checker.root, filepath.FromSlash(rel))); err != nil {
			checker.issue("ERRO", checker.message("Required path missing: ", "Path obrigatório ausente: ")+filepath.Join(checker.root, filepath.FromSlash(rel)))
		}
	}
}

func (checker *nativeChecker) checkSchemaVersion() {
	path := filepath.Join(checker.root, ".pose", "schema-version")
	content, err := os.ReadFile(path)
	if err != nil {
		checker.failOrWarn(checker.message("schema: instance has no .pose/schema-version — run 'pose upgrade'", "schema: instância sem .pose/schema-version — rode 'pose upgrade'"))
		return
	}
	version, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil {
		checker.issue("ERRO", fmt.Sprintf(checker.message("schema: invalid .pose/schema-version (%q)", "schema: .pose/schema-version inválido (%q)"), strings.TrimSpace(string(content))))
		return
	}
	if version > nativeSchemaVersion {
		checker.issue("ERRO", fmt.Sprintf(checker.message("schema: instance v%d is newer than engine v%d", "schema: instância v%d é mais nova que o motor v%d"), version, nativeSchemaVersion))
	}
	if version < nativeSchemaVersion {
		checker.failOrWarn(fmt.Sprintf(checker.message("schema: instance v%d is behind engine v%d — run 'pose upgrade'", "schema: instância v%d atrás do motor v%d — rode 'pose upgrade'"), version, nativeSchemaVersion))
	}
}

func (checker *nativeChecker) checkReferences() {
	for _, rel := range []string{"AGENTS.md", "POSE.md"} {
		path := filepath.Join(checker.root, rel)
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		seen := map[string]bool{}
		for _, match := range checkReference.FindAllStringSubmatch(string(content), -1) {
			ref := strings.TrimRight(match[1], "/")
			if ref == "" || seen[ref] {
				continue
			}
			seen[ref] = true
			if !confinedRelativePath(ref) {
				checker.failOrWarn(fmt.Sprintf(checker.message("Reference escapes the project root: %q (source: %s)", "Referência escapa da raiz do projeto: %q (origem: %s)"), ref, rel))
				continue
			}
			if _, err := os.Stat(filepath.Join(checker.root, filepath.FromSlash(ref))); err != nil {
				checker.failOrWarn(fmt.Sprintf(checker.message("Broken reference: %q (source: %s)", "Referência quebrada: %q (origem: %s)"), ref, rel))
			}
		}
		if len(seen) == 0 {
			checker.issue("ERRO", checker.message("No POSE reference found to validate in ", "Nenhuma referência POSE encontrada para validar em ")+rel)
		}
	}
}

func (checker *nativeChecker) checkValidationMatrix() {
	path := filepath.Join(checker.root, ".pose", "indexes", "validation-matrix.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		checker.failOrWarn(checker.message("validation-matrix.json: invalid JSON: ", "validation-matrix.json: JSON inválido: ") + err.Error())
		return
	}
	allowedTop := map[string]bool{"defaults": true, "stacks": true, "moduleOverrides": true}
	for key := range document {
		if !allowedTop[key] {
			checker.failOrWarn(checker.message("validation-matrix.json: root: unknown key '", "validation-matrix.json: root: chave desconhecida '") + key + "'")
		}
	}
	if defaults, ok := document["defaults"]; ok {
		object, valid := defaults.(map[string]any)
		if !valid {
			checker.failOrWarn(checker.message("validation-matrix.json: defaults: must be an object", "validation-matrix.json: defaults: deve ser objeto"))
		} else if mode, exists := object["mode"]; exists && mode != "strict" && mode != "tolerant" {
			checker.failOrWarn(checker.message("validation-matrix.json: defaults.mode: must be strict or tolerant", "validation-matrix.json: defaults.mode: deve ser strict ou tolerant"))
		}
	}
	checker.validateMatrixObjects(document, "stacks", map[string]bool{"checks": true})
	checker.validateMatrixObjects(document, "moduleOverrides", map[string]bool{"stack": true, "mode": true, "checks": true, "replaceDefaultChecks": true})
	matrix, err := parseValidationMatrix(raw)
	if err != nil {
		checker.failOrWarn("validation-matrix.json: " + err.Error())
		return
	}
	for stackName, stack := range matrix.Stacks {
		for index, check := range stack.Checks {
			checker.validateMatrixCheck(fmt.Sprintf("stacks.%s.checks[%d]", stackName, index), check)
		}
	}
	for module, override := range matrix.ModuleOverrides {
		for index, check := range override.Checks {
			checker.validateMatrixCheck(fmt.Sprintf("moduleOverrides.%s.checks[%d]", module, index), check)
		}
	}
}

func (checker *nativeChecker) validateMatrixObjects(document map[string]any, field string, allowed map[string]bool) {
	value, exists := document[field]
	if !exists {
		return
	}
	objects, ok := value.(map[string]any)
	if !ok {
		checker.failOrWarn("validation-matrix.json: " + field + checker.message(": must be an object", ": deve ser objeto"))
		return
	}
	allowedCheck := map[string]bool{"name": true, "command": true, "program": true, "args": true, "env": true, "severity": true, "when": true}
	for name, rawObject := range objects {
		object, ok := rawObject.(map[string]any)
		if !ok {
			checker.failOrWarn(fmt.Sprintf(checker.message("validation-matrix.json: %s.%s: must be an object", "validation-matrix.json: %s.%s: deve ser objeto"), field, name))
			continue
		}
		for key := range object {
			if !allowed[key] {
				checker.failOrWarn(fmt.Sprintf(checker.message("validation-matrix.json: %s.%s: unknown key '%s'", "validation-matrix.json: %s.%s: chave desconhecida '%s'"), field, name, key))
			}
		}
		checks, exists := object["checks"]
		if !exists {
			continue
		}
		list, ok := checks.([]any)
		if !ok {
			checker.failOrWarn(fmt.Sprintf(checker.message("validation-matrix.json: %s.%s.checks: must be a list", "validation-matrix.json: %s.%s.checks: deve ser lista"), field, name))
			continue
		}
		for index, rawCheck := range list {
			check, ok := rawCheck.(map[string]any)
			if !ok {
				checker.failOrWarn(fmt.Sprintf(checker.message("validation-matrix.json: %s.%s.checks[%d]: must be an object", "validation-matrix.json: %s.%s.checks[%d]: deve ser objeto"), field, name, index))
				continue
			}
			for key := range check {
				if !allowedCheck[key] {
					checker.failOrWarn(fmt.Sprintf(checker.message("validation-matrix.json: %s.%s.checks[%d]: unknown key '%s'", "validation-matrix.json: %s.%s.checks[%d]: chave desconhecida '%s'"), field, name, index, key))
				}
			}
		}
	}
}

func (checker *nativeChecker) validateMatrixCheck(prefix string, check validationCheck) {
	if strings.TrimSpace(check.Name) == "" {
		checker.failOrWarn(prefix + checker.message(".name: required", ".name: obrigatório"))
	}
	if (check.Command == "") == (check.Program == "") {
		checker.failOrWarn(prefix + checker.message(": requires exactly one of command or program", ": exige exatamente um de command ou program"))
	}
	if check.Command != "" && len(check.Args) > 0 {
		checker.failOrWarn(prefix + checker.message(".args: accepted only with structured program", ".args: só é aceito com program estruturado"))
	}
	if check.Severity != "" && check.Severity != "required" && check.Severity != "optional" {
		checker.failOrWarn(prefix + checker.message(".severity: invalid", ".severity: inválida"))
	}
	for key := range check.Env {
		if strings.TrimSpace(key) == "" || strings.Contains(key, "=") {
			checker.failOrWarn(prefix + checker.message(".env: invalid key", ".env: chave inválida"))
		}
	}
	for field, path := range map[string]string{"fileExists": check.When.FileExists, "fileNotExists": check.When.FileNotExists} {
		if !confinedRelativePath(path) {
			checker.failOrWarn(prefix + ".when." + field + checker.message(": must remain inside its module", ": deve permanecer dentro do módulo"))
		}
	}
}

func (checker *nativeChecker) checkTaskMap() {
	path := filepath.Join(checker.root, ".pose", "indexes", "task-map.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var rawDocument map[string]any
	if err := json.Unmarshal(raw, &rawDocument); err != nil {
		checker.failOrWarn(checker.message("task-map.json: invalid JSON: ", "task-map.json: JSON inválido: ") + err.Error())
		return
	}
	if _, ok := rawDocument["tasks"].(map[string]any); !ok {
		checker.failOrWarn(checker.message("task-map.json: tasks: must be an object", "task-map.json: tasks: deve ser objeto"))
		return
	}
	var document struct {
		Tasks map[string]struct {
			Workflow, Skill string
			Rules           []string
		} `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		checker.failOrWarn(checker.message("task-map.json: invalid schema: ", "task-map.json: schema inválido: ") + err.Error())
		return
	}
	for name, task := range document.Tasks {
		if task.Workflow != "" {
			checker.requireTaskRef(name+".workflow", task.Workflow)
		}
		if task.Skill != "" {
			checker.requireTaskRef(name+".skill", filepath.ToSlash(filepath.Join(".agents", "skills", task.Skill, "SKILL.md")))
		}
		for _, rule := range task.Rules {
			checker.requireTaskRef(name+".rules", filepath.ToSlash(filepath.Join(".pose", "rules", rule+".md")))
		}
	}
}

func (checker *nativeChecker) requireTaskRef(field, rel string) {
	if !confinedRelativePath(rel) {
		checker.failOrWarn(fmt.Sprintf(checker.message("task-map.json: %s escapes the project root: %s", "task-map.json: %s escapa da raiz do projeto: %s"), field, rel))
		return
	}
	if _, err := os.Stat(filepath.Join(checker.root, filepath.FromSlash(rel))); err != nil {
		checker.failOrWarn(fmt.Sprintf(checker.message("task-map.json: %s does not exist: %s", "task-map.json: %s inexistente: %s"), field, rel))
	}
}

type checkSpec struct {
	slug, status, dependsOn, path string
	priority                      int
}

func (checker *nativeChecker) checkSpecs() {
	paths, _ := filepath.Glob(filepath.Join(checker.root, ".pose", "specs", "*", "spec.md"))
	legacy, _ := filepath.Glob(filepath.Join(checker.root, ".pose", "specs", "*.md"))
	for _, path := range legacy {
		if !strings.EqualFold(filepath.Base(path), "README.md") {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	specs := map[string]checkSpec{}
	validStatus := map[string]bool{"draft": true, "in-progress": true, "done": true, "blocked": true, "superseded": true, "abandoned": true}
	for _, path := range paths {
		fields := simpleFrontmatter(path)
		slug := filepath.Base(filepath.Dir(path))
		if filepath.Dir(path) == filepath.Join(checker.root, ".pose", "specs") {
			slug = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
		status := fields["status"]
		if status != "" && !validStatus[status] {
			checker.failOrWarn(fmt.Sprintf(checker.message("spec status: %s: invalid status: %q", "spec status: %s: status inválido: %q"), slug, status))
		}
		priority := 0
		if fields["priority"] != "" {
			parsed, err := strconv.Atoi(fields["priority"])
			if err != nil || parsed < 0 {
				checker.failOrWarn(fmt.Sprintf(checker.message("spec deps: %s: invalid priority", "spec deps: %s: priority inválida"), slug))
			} else {
				priority = parsed
			}
		}
		specs[slug] = checkSpec{slug, status, fields["depends_on"], path, priority}
	}
	edges := map[string][]string{}
	for slug, spec := range specs {
		seen := map[string]bool{}
		for _, ref := range splitInlineList(spec.dependsOn) {
			if seen[ref] {
				checker.issue("AVISO", fmt.Sprintf(checker.message("spec deps: %s: duplicate dependency: %s", "spec deps: %s: dependência duplicada: %s"), slug, ref))
				continue
			}
			seen[ref] = true
			switch {
			case checkSlug.MatchString(ref):
				if ref == slug {
					checker.failOrWarn(fmt.Sprintf(checker.message("spec deps: %s: depends on itself", "spec deps: %s: depende da própria spec"), slug))
				} else if _, ok := specs[ref]; !ok {
					checker.failOrWarn(fmt.Sprintf(checker.message("spec deps: %s: missing spec: %s", "spec deps: %s: spec inexistente: %s"), slug, ref))
				} else {
					edges[slug] = append(edges[slug], ref)
				}
			case checkMilestoneRef.MatchString(ref), checkRoadmapRef.MatchString(ref):
			default:
				checker.failOrWarn(fmt.Sprintf(checker.message("spec deps: %s: invalid reference: %s", "spec deps: %s: ref inválida: %s"), slug, ref))
			}
		}
	}
	visiting, visited := map[string]bool{}, map[string]bool{}
	var visit func(string) bool
	visit = func(slug string) bool {
		if visiting[slug] {
			return true
		}
		if visited[slug] {
			return false
		}
		visiting[slug] = true
		for _, dep := range edges[slug] {
			if visit(dep) {
				return true
			}
		}
		delete(visiting, slug)
		visited[slug] = true
		return false
	}
	for slug := range specs {
		if visit(slug) {
			checker.failOrWarn(checker.message("spec deps: cycle detected involving ", "spec deps: ciclo detectado envolvendo ") + slug)
			break
		}
	}
	checker.checkRoadmaps(specs)
}

type checkMilestone struct {
	id, targetStart, targetDue string
	after, specs               []string
}

type checkRoadmap struct {
	slug, status string
	dependsOn    []string
	milestones   []checkMilestone
}

func parseRoadmap(path string) checkRoadmap {
	fields := simpleFrontmatter(path)
	roadmap := checkRoadmap{slug: fields["slug"], status: fields["status"], dependsOn: splitInlineList(fields["depends_on"])}
	if roadmap.slug == "" {
		roadmap.slug = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return roadmap
	}
	current := -1
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## Milestone:") {
			roadmap.milestones = append(roadmap.milestones, checkMilestone{id: strings.TrimSpace(strings.TrimPrefix(trimmed, "## Milestone:"))})
			current = len(roadmap.milestones) - 1
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			current = -1
			continue
		}
		if current < 0 || !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		key, value, found := strings.Cut(strings.TrimPrefix(trimmed, "- "), ":")
		if !found {
			continue
		}
		value = strings.TrimSpace(strings.SplitN(value, "#", 2)[0])
		switch strings.TrimSpace(key) {
		case "after":
			roadmap.milestones[current].after = splitInlineList(value)
		case "specs":
			roadmap.milestones[current].specs = splitInlineList(value)
		case "target_start":
			roadmap.milestones[current].targetStart = value
		case "target_due":
			roadmap.milestones[current].targetDue = value
		}
	}
	return roadmap
}

func (checker *nativeChecker) checkRoadmaps(specs map[string]checkSpec) {
	paths, _ := filepath.Glob(filepath.Join(checker.root, ".pose", "roadmaps", "*.md"))
	roadmaps := map[string]checkRoadmap{}
	for _, path := range paths {
		if strings.EqualFold(filepath.Base(path), "README.md") {
			continue
		}
		rm := parseRoadmap(path)
		roadmaps[rm.slug] = rm
	}
	owners := map[string]string{}
	milestones := map[string]bool{}
	roadmapEdges := map[string][]string{}
	validRoadmapStatus := map[string]bool{"draft": true, "active": true, "done": true, "abandoned": true}
	datePattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	for slug, rm := range roadmaps {
		if rm.status != "" && !validRoadmapStatus[rm.status] {
			checker.failOrWarn(fmt.Sprintf(checker.message("spec deps: roadmap %s: invalid status: %s", "spec deps: roadmap %s: status inválido: %s"), slug, rm.status))
		}
		for _, dep := range rm.dependsOn {
			if dep == slug {
				checker.failOrWarn(fmt.Sprintf(checker.message("spec deps: roadmap %s depends on itself", "spec deps: roadmap %s depende de si próprio"), slug))
			} else if _, ok := roadmaps[dep]; !ok {
				checker.failOrWarn(fmt.Sprintf(checker.message("spec deps: roadmap %s references missing roadmap: %s", "spec deps: roadmap %s referencia roadmap inexistente: %s"), slug, dep))
			} else {
				roadmapEdges[slug] = append(roadmapEdges[slug], dep)
			}
		}
		seen := map[string]bool{}
		for _, milestone := range rm.milestones {
			if !checkSlug.MatchString(milestone.id) {
				checker.failOrWarn(fmt.Sprintf(checker.message("spec deps: roadmap %s: invalid milestone: %s", "spec deps: roadmap %s: milestone inválido: %s"), slug, milestone.id))
				continue
			}
			if seen[milestone.id] {
				checker.failOrWarn(fmt.Sprintf(checker.message("spec deps: roadmap %s: duplicate milestone: %s", "spec deps: roadmap %s: milestone duplicado: %s"), slug, milestone.id))
			}
			seen[milestone.id] = true
			milestones[slug+"/"+milestone.id] = true
			for name, value := range map[string]string{"target_start": milestone.targetStart, "target_due": milestone.targetDue} {
				if value != "" && !datePattern.MatchString(value) {
					checker.failOrWarn(fmt.Sprintf(checker.message("spec deps: roadmap %s/%s: invalid %s: %s", "spec deps: roadmap %s/%s: %s inválido: %s"), slug, milestone.id, name, value))
				}
			}
			if milestone.targetStart != "" && milestone.targetDue != "" && milestone.targetStart > milestone.targetDue {
				checker.failOrWarn(fmt.Sprintf("spec deps: roadmap %s/%s: target_start > target_due", slug, milestone.id))
			}
			for _, spec := range milestone.specs {
				if _, ok := specs[spec]; !ok {
					checker.failOrWarn(fmt.Sprintf(checker.message("spec deps: roadmap %s/%s: missing spec: %s", "spec deps: roadmap %s/%s: spec inexistente: %s"), slug, milestone.id, spec))
				} else if rm.status == "active" && owners[spec] != "" && owners[spec] != slug {
					checker.failOrWarn(fmt.Sprintf(checker.message("spec deps: spec %s belongs to two active roadmaps: %s and %s", "spec deps: spec %s pertence a dois roadmaps ativos: %s e %s"), spec, owners[spec], slug))
				} else if rm.status == "active" {
					owners[spec] = slug
				}
			}
		}
		milestoneEdges := map[string][]string{}
		for _, milestone := range rm.milestones {
			for _, dep := range milestone.after {
				if strings.HasPrefix(dep, "spec:") {
					if _, ok := specs[strings.TrimPrefix(dep, "spec:")]; !ok {
						checker.failOrWarn(fmt.Sprintf(checker.message("spec deps: roadmap %s/%s: after references a missing spec: %s", "spec deps: roadmap %s/%s: after referencia spec inexistente: %s"), slug, milestone.id, dep))
					}
				} else if !seen[dep] {
					checker.failOrWarn(fmt.Sprintf(checker.message("spec deps: roadmap %s/%s: after references a missing milestone: %s", "spec deps: roadmap %s/%s: after referencia milestone inexistente: %s"), slug, milestone.id, dep))
				} else {
					milestoneEdges[milestone.id] = append(milestoneEdges[milestone.id], dep)
				}
			}
		}
		if graphHasCycle(milestoneEdges) {
			checker.failOrWarn(checker.message("spec deps: milestone cycle in roadmap ", "spec deps: ciclo entre milestones do roadmap ") + slug)
		}
	}
	if graphHasCycle(roadmapEdges) {
		checker.failOrWarn(checker.message("spec deps: dependency cycle between roadmaps", "spec deps: ciclo de dependência entre roadmaps"))
	}
	if len(roadmaps) > 0 {
		for slug, spec := range specs {
			for _, ref := range splitInlineList(spec.dependsOn) {
				if strings.HasPrefix(ref, "roadmap:") {
					if _, ok := roadmaps[strings.TrimPrefix(ref, "roadmap:")]; !ok {
						checker.failOrWarn(fmt.Sprintf(checker.message("spec deps: %s references a missing roadmap: %s", "spec deps: %s referencia roadmap inexistente: %s"), slug, ref))
					}
				} else if strings.HasPrefix(ref, "milestone:") && !milestones[strings.TrimPrefix(ref, "milestone:")] {
					checker.failOrWarn(fmt.Sprintf(checker.message("spec deps: %s references a missing milestone: %s", "spec deps: %s referencia milestone inexistente: %s"), slug, ref))
				}
			}
		}
	}
}

func graphHasCycle(edges map[string][]string) bool {
	visiting, visited := map[string]bool{}, map[string]bool{}
	var visit func(string) bool
	visit = func(node string) bool {
		if visiting[node] {
			return true
		}
		if visited[node] {
			return false
		}
		visiting[node] = true
		for _, dep := range edges[node] {
			if visit(dep) {
				return true
			}
		}
		delete(visiting, node)
		visited[node] = true
		return false
	}
	for node := range edges {
		if visit(node) {
			return true
		}
	}
	return false
}

func (checker *nativeChecker) checkChangelogs() {
	paths, _ := filepath.Glob(filepath.Join(checker.root, ".pose", "changelogs", "unreleased", "*.md"))
	valid := map[string]bool{"added": true, "changed": true, "fixed": true, "removed": true, "deprecated": true, "security": true}
	covered := map[string]bool{}
	for _, path := range paths {
		if strings.EqualFold(filepath.Base(path), "README.md") {
			continue
		}
		fields := simpleFrontmatter(path)
		slug := strings.TrimSuffix(filepath.Base(path), ".md")
		if fields["spec"] != "" {
			slug = fields["spec"]
		}
		covered[slug] = true
		if _, err := os.Stat(filepath.Join(checker.root, ".pose", "specs", slug, "spec.md")); err != nil {
			checker.failOrWarn(fmt.Sprintf(checker.message("changelog: fragment %s points to a missing spec: %s", "changelog: fragment %s aponta para spec inexistente: %s"), filepath.Base(path), slug))
		}
		if !valid[fields["category"]] {
			checker.failOrWarn(fmt.Sprintf(checker.message("changelog: fragment %s has an invalid category", "changelog: fragment %s tem category inválida"), filepath.Base(path)))
		}
		if strings.TrimSpace(stripHTMLComments(frontmatterBody(path))) == "" {
			checker.failOrWarn(fmt.Sprintf(checker.message("changelog: fragment %s has an empty body", "changelog: fragment %s tem corpo vazio"), filepath.Base(path)))
		}
	}
	released, _ := filepath.Glob(filepath.Join(checker.root, ".pose", "changelogs", "*.md"))
	for _, path := range released {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		text := string(raw)
		specPaths, _ := filepath.Glob(filepath.Join(checker.root, ".pose", "specs", "*", "spec.md"))
		for _, specPath := range specPaths {
			slug := filepath.Base(filepath.Dir(specPath))
			if strings.Contains(text, slug) {
				covered[slug] = true
			}
		}
	}
	policyPath := filepath.Join(checker.root, ".pose", "policy", "changelog.json")
	policyRaw, err := os.ReadFile(policyPath)
	if err != nil {
		return
	}
	var policy struct {
		AdoptedAt string `json:"adopted_at"`
	}
	if json.Unmarshal(policyRaw, &policy) != nil || policy.AdoptedAt == "" {
		return
	}
	specPaths, _ := filepath.Glob(filepath.Join(checker.root, ".pose", "specs", "*", "spec.md"))
	for _, path := range specPaths {
		fields := simpleFrontmatter(path)
		slug := filepath.Base(filepath.Dir(path))
		if fields["status"] == "done" && fields["changelog"] != "none" && fields["completed_at"] >= policy.AdoptedAt && !covered[slug] {
			checker.issue("AVISO", checker.message("changelog: done spec without a changelog fragment: ", "changelog: spec done sem changelog fragment: ")+slug)
		}
	}
}

func frontmatterBody(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(raw), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return string(raw)
	}
	for index, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			return strings.Join(lines[index+2:], "\n")
		}
	}
	return string(raw)
}

func stripHTMLComments(value string) string {
	return regexp.MustCompile(`(?s)<!--.*?-->`).ReplaceAllString(value, "")
}

func (checker *nativeChecker) checkReadyTransitions() {
	if _, err := exec.Command("git", "-C", checker.root, "rev-parse", "--verify", "HEAD").Output(); err != nil {
		return
	}
	changed := map[string]bool{}
	for _, command := range [][]string{
		{"git", "-C", checker.root, "diff", "--name-only", "HEAD", "--", ".pose/specs"},
		{"git", "-C", checker.root, "ls-files", "--others", "--exclude-standard", "--", ".pose/specs"},
	} {
		output, err := exec.Command(command[0], command[1:]...).Output()
		if err != nil {
			continue
		}
		for _, rel := range strings.Fields(string(output)) {
			if strings.HasPrefix(filepath.ToSlash(rel), ".pose/specs/") && strings.HasSuffix(rel, "/spec.md") {
				changed[filepath.ToSlash(rel)] = true
			}
		}
	}
	for rel := range changed {
		path := filepath.Join(checker.root, filepath.FromSlash(rel))
		if simpleFrontmatter(path)["status"] != "in-progress" {
			continue
		}
		old, _ := exec.Command("git", "-C", checker.root, "show", "HEAD:"+rel).Output()
		oldStatus := frontmatterFromText(string(old))["status"]
		if oldStatus == "in-progress" {
			continue
		}
		if !specReady(checker.root, path) {
			slug := filepath.Base(filepath.Dir(path))
			checker.failOrWarn(checker.message("DoR: transition to in-progress without Definition of Ready: ", "DoR: transição para in-progress sem Definition of Ready: ") + slug + checker.message(" (details: pose lint-spec ", " (detalhes: pose lint-spec ") + slug + " --ready-check)")
		}
	}
}

func specReady(root, path string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	fields := frontmatterFromText(string(raw))
	taskType := fields["task_type"]
	if taskType == "" {
		taskType = "feature"
	}
	required := []string{"Intent", "Requirements", "Technical Plan"}
	policyRaw, err := os.ReadFile(filepath.Join(root, ".pose", "policy", "dor.json"))
	if err == nil {
		var policy struct {
			DefaultTaskType string              `json:"defaultTaskType"`
			TaskTypes       map[string][]string `json:"taskTypes"`
		}
		if json.Unmarshal(policyRaw, &policy) == nil {
			if fields["task_type"] == "" && policy.DefaultTaskType != "" {
				taskType = policy.DefaultTaskType
			}
			if configured, ok := policy.TaskTypes[taskType]; ok {
				required = configured
			} else if configured, ok := policy.TaskTypes[policy.DefaultTaskType]; ok {
				taskType, required = policy.DefaultTaskType, configured
			}
		}
	}
	sections := specSections(stripHTMLComments(string(raw)))
	for _, name := range required {
		if !sectionFilled(sections[name]) {
			return false
		}
	}
	if taskType == "feature" {
		criterion := regexp.MustCompile(`(?m)^\s*-\s*R\d+\s*(?:\[[^]]+\])?\s*[:—-]\s*\S`)
		if !criterion.MatchString(strings.Join(sections["Requirements"], "\n")) {
			return false
		}
	}
	for _, ref := range splitInlineList(fields["depends_on"]) {
		if !checkSlug.MatchString(ref) && !checkMilestoneRef.MatchString(ref) && !checkRoadmapRef.MatchString(ref) {
			return false
		}
	}
	return true
}

func specSections(text string) map[string][]string {
	sections := map[string][]string{}
	canonical := regexp.MustCompile(`^##\s+(?:\d+\.\s*)?(Intent|Requirements|Technical Plan|Tasks|Decisions|Validation|Final Report)\s*$`)
	current := ""
	for _, line := range strings.Split(text, "\n") {
		if match := canonical.FindStringSubmatch(strings.TrimSpace(line)); match != nil {
			current = match[1]
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "## ") {
			current = ""
			continue
		}
		if current != "" {
			sections[current] = append(sections[current], line)
		}
	}
	return sections
}

func sectionFilled(lines []string) bool {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || trimmed == "-" || trimmed == "- [ ]" {
			continue
		}
		if strings.HasPrefix(trimmed, "> Definition of Ready") || strings.HasPrefix(trimmed, "> Published IDs") || strings.HasPrefix(trimmed, "> Optional EARS") {
			continue
		}
		return true
	}
	return false
}

func simpleFrontmatter(path string) map[string]string {
	fields := map[string]string{}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fields
	}
	return frontmatterFromText(string(raw))
}

func frontmatterFromText(text string) map[string]string {
	fields := map[string]string{}
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return fields
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		if strings.HasPrefix(strings.TrimSpace(line), "#") || !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		fields[strings.TrimSpace(parts[0])] = strings.TrimSpace(strings.SplitN(parts[1], "#", 2)[0])
	}
	return fields
}

func splitInlineList(value string) []string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		value = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
	}
	result := []string{}
	for _, item := range strings.Split(value, ",") {
		if strings.TrimSpace(item) != "" {
			result = append(result, strings.TrimSpace(item))
		}
	}
	sort.Strings(result)
	return result
}

// checkCapabilities runs the capability-assessment validation (spec
// pose-capability-mechanism, R7) when the opt-in artifact exists. Absence is
// not an issue; a present artifact must validate, and staleness surfaces as
// warnings regardless of mode.
func (checker *nativeChecker) checkCapabilities() {
	store := pose.Store{Root: checker.root}
	if !store.HasCapabilityAssessment() {
		return
	}
	report, err := runAssessValidation(checker.root)
	if err != nil {
		checker.failOrWarn(checker.message("capabilities: ", "capabilities: ") + err.Error())
		return
	}
	for _, issue := range report.Errors {
		checker.failOrWarn(checker.message("capabilities: ", "capabilities: ") + issue)
	}
	for _, warning := range report.Warnings {
		checker.issue("AVISO", checker.message("capabilities: ", "capabilities: ")+warning)
	}
}
