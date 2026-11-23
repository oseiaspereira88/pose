package cli

// Carried checks (spec pose-validate-carries-unchanged-checks): with a
// comparable canonical result as base, validate --carry runs the modules the
// changed scope selects between the base head and HEAD and carries the base
// results of every other module, each stating the head it ran at. Any doubt
// about the base runs everything and records why (R2).

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"strings"
	"unicode"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/version"
)

// carryRecord is the run's account of its base (R5).
type carryRecord struct {
	Used     bool              `json:"used"`
	Reason   string            `json:"reason,omitempty"`
	BaseHead string            `json:"base_head,omitempty"`
	Changed  int               `json:"changed"`
	Selected map[string]string `json:"selected,omitempty"`
}

// carryPlan is what a usable base lets the run reuse.
type carryPlan struct {
	record    *carryRecord
	selection *scopeSelection
	base      map[string]checkResult
	baseHead  string
	baseTime  string
}

// carried reports whether module's checks come from the base.
func (p carryPlan) carried(module string) bool {
	if p.selection == nil {
		return false
	}
	_, selected := p.selection.Selected[module]
	return !selected
}

// carriedResult is the base result of a check as this run records it (R4):
// unchanged, plus where and when it really ran, kept through chained carries.
func (p carryPlan) carriedResult(id string) checkResult {
	result := p.base[id]
	if result.CarriedFrom == "" {
		result.CarriedFrom, result.CarriedGeneratedAt = p.baseHead, p.baseTime
	}
	return result
}

// planCarry judges the canonical result at resultsPath as a base. A nil
// selection means every check runs; the record says why.
func planCarry(root, resultsPath, mode, matrixSHA, head string, matrix validationMatrix, modules []validationModule) carryPlan {
	plan := carryPlan{record: &carryRecord{}}
	engine, engineVerifiable := engineIdentityFrom(debug.ReadBuildInfo())
	unused := func(format string, args ...any) carryPlan {
		plan.record.Reason = fmt.Sprintf(format, args...)
		return plan
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(resultsPath)))
	if err != nil {
		return unused("no canonical result at %s", resultsPath)
	}
	var base validationRunResult
	if err := json.Unmarshal(raw, &base); err != nil || base.SchemaVersion != validationResultSchema {
		return unused("the canonical result is unreadable or has another schema")
	}
	switch {
	case base.ModuleFilter != "" || base.StackFilter != "":
		return unused("the canonical result was filtered by module or stack, so it is not comparable")
	case base.Mode != mode:
		return unused("the canonical result ran in mode %q, this run is %q", base.Mode, mode)
	case base.MatrixSHA256 != matrixSHA:
		return unused("the validation matrix changed since the canonical result")
	case !engineVerifiable:
		return unused("this engine build was modified from its recorded revision, so its identity cannot be compared")
	case base.EngineVersion == "" || base.EngineVersion != engine || strings.HasSuffix(base.EngineVersion, ".dirty"):
		return unused("the engine version differs from the canonical result's or was not recorded")
	case base.GitHead == "" || !safeGitRevRE.MatchString(base.GitHead):
		return unused("the canonical result records no usable head")
	case head == "" || !safeGitRevRE.MatchString(head):
		return unused("the current head cannot be read")
	}
	for _, check := range base.Checks {
		if check.LimitState == "cancelled" {
			return unused("the canonical result was cancelled")
		}
	}
	// The base must have validated the commit it records: a run over a working
	// copy that differed from its head proves nothing about that head.
	if !base.MaterialClean {
		return unused("the canonical result did not validate a clean working copy of its head")
	}
	for _, binding := range base.MaterialProvenance {
		if binding == posemodel.MaterialBindingUnavailable {
			return unused("the canonical result records an unavailable material binding")
		}
	}
	if !materialWorktreeClean(root) {
		return unused("the working copy differs from HEAD in material paths")
	}
	// Selection is lexical: a module reached through a symlink aliases another
	// path, so its inputs cannot be told apart (review round 4, N9).
	// Two modules sharing one physical directory (a case alias on a
	// case-insensitive filesystem) alias each other too (review round 5, N13).
	if reason := aliasedModule(modules, filepath.EvalSymlinks, os.Stat); reason != "" {
		return unused("%s", reason)
	}
	// Lexical selection cannot see through aliases. Rather than chase every
	// alias form (review rounds 4–7, N9, N13, N16–N18), a carry needs a
	// case-sensitive filesystem and no tracked symlink that links a module
	// (other than the root) or a declared dependency with any other path.
	meta, err := loadScopeMetadata(root)
	if err != nil {
		return unused("changed-scope selection failed: %v", err)
	}
	// Symlinks inside a submodule hide behind its gitlink (review round 8,
	// N20): scan each checked-out submodule at its recorded commit, and refuse
	// when a module or a dependency lies in one that cannot be scanned.
	if reason, err := submoduleAliases(root, head, modules, meta); err != nil {
		return unused("submodules cannot be read: %v", err)
	} else if reason != "" {
		return unused("%s", reason)
	}
	if link, err := aliasingSymlink(root, head, modules, meta); err != nil {
		return unused("tracked symlinks cannot be read: %v", err)
	} else if link != "" {
		return unused("the tree tracks symlink %s into or out of a module or a declared dependency, so changes cannot be attributed", link)
	}
	if caseInsensitiveFilesystem(root, os.Stat) {
		return unused("the filesystem is case-insensitive, so changes cannot be attributed to modules")
	}
	// A case-folded directory inside a case-sensitive filesystem is an alias
	// too (review round 8, N22).
	for _, dir := range carryRelevantDirs(root, modules, meta) {
		if casefoldedDirectory(dir, os.ReadDir, os.Stat) {
			return unused("directory %s folds case, so changes cannot be attributed to modules", dir)
		}
	}
	// Check ids must be unique: two checks sharing an id would share one base
	// result (review round 7, N19).
	if id := duplicateCheckID(matrix, modules); id != "" {
		return unused("the matrix defines check %s more than once", id)
	}
	baseIDs := map[string]bool{}
	for _, check := range base.Checks {
		if baseIDs[check.ID] {
			return unused("the canonical result records check %s more than once", check.ID)
		}
		baseIDs[check.ID] = true
	}
	if err := posemodel.RawGit("-C", root, "merge-base", "--is-ancestor", base.GitHead, head).Run(); err != nil {
		return unused("the canonical result's head %s is not an ancestor of HEAD", base.GitHead)
	}
	selection, err := computeChangedScope(root, base.GitHead, head, modules, carryExclusions(root, base.GitHead, head, resultsPath)...)
	if err != nil {
		return unused("changed-scope selection failed: %v", err)
	}
	plan.base = map[string]checkResult{}
	for _, check := range base.Checks {
		plan.base[check.ID] = check
	}
	// A module whose checks are not all in the base runs (R3).
	for _, module := range modules {
		if _, selected := selection.Selected[module.Rel]; selected {
			continue
		}
		stack, checks, _ := moduleValidationChecks(module, matrix, "", "")
		for _, check := range checks {
			if check.Program == "" {
				continue
			}
			id := module.Rel + "/" + stack + "/" + check.Name
			if prior, ok := plan.base[id]; !ok || prior.Outcome == "skipped" || prior.Outcome == "" {
				selection.Selected[module.Rel] = "carry: no prior result for " + id
				break
			}
		}
	}
	plan.selection, plan.baseHead, plan.baseTime = &selection, base.GitHead, base.GeneratedAt
	plan.record.Used, plan.record.BaseHead, plan.record.Changed, plan.record.Selected = true, base.GitHead, len(selection.Changed), selection.Selected
	return plan
}

// carryExclusions are the changed paths that are not changes for the carry
// selection (R3): the canonical result itself and the derived administrative
// records of the engine's closed list (the same classes material bindings
// ignore) that are still regular files, or gone, at HEAD. A symlink,
// submodule or directory in a class stays a change.
func carryExclusions(root, base, head, resultsPath string) []string {
	exclude := []string{resultsPath}
	out, err := posemodel.RawGit("-C", root, "-c", "diff.ignoreSubmodules=none", "diff", "--ignore-submodules=none", "--name-only", "-z", "--no-renames", base, head, "--").Output()
	if err != nil {
		return exclude
	}
	for _, file := range strings.Split(string(out), "\x00") {
		file = filepath.ToSlash(file)
		if file == "" || !posemodel.IsAdministrativePath(file) {
			continue
		}
		entry, err := posemodel.RawGit("-C", root, "ls-tree", "-z", head, "--", file).Output()
		if err != nil {
			continue
		}
		fields := strings.Fields(string(entry))
		if len(fields) == 0 || fields[0] == "100644" || fields[0] == "100755" {
			exclude = append(exclude, file)
		}
	}
	return exclude
}

// validationEngineIdentity names the engine that ran a validation: its version
// plus, when the binary records it, the VCS revision it was built from, marked
// when the build tree was modified. Development builds share a version string,
// so the revision is what tells two of them apart (R2).
func validationEngineIdentity() string {
	identity, _ := engineIdentityFrom(debug.ReadBuildInfo())
	return identity
}

// engineIdentityFrom builds the identity from build information. A build of a
// modified tree is not verifiable: two such builds at one revision share an
// identity while their code differs (review round 2, N4).
func engineIdentityFrom(info *debug.BuildInfo, ok bool) (string, bool) {
	identity := version.Version
	if !ok || info == nil {
		return identity, true
	}
	revision, modified := "", false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	if revision != "" {
		identity += "+" + revision
		if modified {
			identity += ".dirty"
		}
	}
	return identity, !modified
}

// aliasedModule names a module whose changes cannot be attributed lexically:
// one reached through a symlink, or one sharing its physical directory with
// another module, as a case alias does on a case-insensitive filesystem
// (review rounds 4–5, N9 and N13).
func aliasedModule(modules []validationModule, resolve func(string) (string, error), stat func(string) (os.FileInfo, error)) string {
	seen := make([]os.FileInfo, 0, len(modules))
	for _, module := range modules {
		resolved, err := resolve(module.Abs)
		if err != nil || filepath.Clean(resolved) != filepath.Clean(module.Abs) {
			return fmt.Sprintf("module %s is reached through a symlink, so its changes cannot be attributed", module.Rel)
		}
		info, err := stat(module.Abs)
		if err != nil {
			return fmt.Sprintf("module %s cannot be read", module.Rel)
		}
		for _, other := range seen {
			if os.SameFile(info, other) {
				return fmt.Sprintf("module %s shares its directory with another module, so its changes cannot be attributed", module.Rel)
			}
		}
		seen = append(seen, info)
	}
	return ""
}

// aliasingSymlink returns a tracked symlink that could hide a change from the
// lexical selection: one whose path or target lies in a module other than the
// root, or intersects a declared dependency, or whose target leaves the
// repository. Symlinks between paths no module owns are harmless: a change on
// either side already selects the root module and every module that is not
// self-contained.
func aliasingSymlink(root, head string, modules []validationModule, meta scopeMetadata) (string, error) {
	return aliasingSymlinkIn(root, root, head, "", modules, meta)
}

// aliasingSymlinkIn scans the tree of the repository at repo, at commit, whose
// paths sit under prefix in the project.
func aliasingSymlinkIn(root, repo, commit, prefix string, modules []validationModule, meta scopeMetadata) (string, error) {
	out, err := posemodel.RawGit("-C", repo, "ls-tree", "-r", "-z", commit).Output()
	if err != nil {
		return "", err
	}
	var deps []string
	for _, module := range meta.Modules {
		for _, dep := range module.DependsOn {
			deps = append(deps, filepath.ToSlash(filepath.Clean(dep)))
		}
	}
	touches := func(p string) bool {
		for _, m := range modules {
			if m.Rel != "." && (p == m.Rel || strings.HasPrefix(p, m.Rel+"/") || strings.HasPrefix(m.Rel, p+"/")) {
				return true
			}
		}
		for _, dep := range deps {
			if dep == "." || p == dep || strings.HasPrefix(p, dep+"/") || strings.HasPrefix(dep, p+"/") {
				return true
			}
		}
		return false
	}
	for _, entry := range strings.Split(string(out), "\x00") {
		if !strings.HasPrefix(entry, "120000 ") {
			continue
		}
		tab := strings.IndexByte(entry, '\t')
		if tab < 0 {
			return "", fmt.Errorf("unexpected tree entry %q", entry)
		}
		fields := strings.Fields(entry[:tab])
		if len(fields) != 3 {
			return "", fmt.Errorf("unexpected tree entry %q", entry)
		}
		link := path.Join(prefix, entry[tab+1:])
		blob, err := posemodel.RawGit("-C", repo, "cat-file", "blob", fields[2]).Output()
		if err != nil {
			return "", err
		}
		target := string(blob)
		resolved := path.Clean(path.Join(path.Dir(link), target))
		if path.IsAbs(target) || resolved == ".." || strings.HasPrefix(resolved, "../") || touches(link) || touches(resolved) {
			return link, nil
		}
	}
	return "", nil
}

// caseInsensitiveFilesystem reports whether root's filesystem folds case: the
// upper-case spelling of its .git entry names the same file.
func caseInsensitiveFilesystem(root string, stat func(string) (os.FileInfo, error)) bool {
	lower, err := stat(filepath.Join(root, ".git"))
	if err != nil {
		return true
	}
	upper, err := stat(filepath.Join(root, ".GIT"))
	return err == nil && os.SameFile(lower, upper)
}

// duplicateCheckID returns a check id the matrix defines more than once.
func duplicateCheckID(matrix validationMatrix, modules []validationModule) string {
	seen := map[string]bool{}
	for _, module := range modules {
		stack, checks, _ := moduleValidationChecks(module, matrix, "", "")
		for _, check := range checks {
			if check.Program == "" {
				continue
			}
			id := module.Rel + "/" + stack + "/" + check.Name
			if seen[id] {
				return id
			}
			seen[id] = true
		}
	}
	return ""
}

// gitlinks lists the gitlinks of a tree: path and recorded commit.
func gitlinks(repo, commit string) (map[string]string, error) {
	out, err := posemodel.RawGit("-C", repo, "ls-tree", "-r", "-z", commit).Output()
	if err != nil {
		return nil, err
	}
	links := map[string]string{}
	for _, entry := range strings.Split(string(out), "\x00") {
		if !strings.HasPrefix(entry, "160000 ") {
			continue
		}
		tab := strings.IndexByte(entry, '\t')
		fields := strings.Fields(entry[:max(tab, 0)])
		if tab < 0 || len(fields) != 3 {
			return nil, fmt.Errorf("unexpected tree entry %q", entry)
		}
		links[entry[tab+1:]] = fields[2]
	}
	return links, nil
}

// submoduleAliases names a submodule that could hide an alias: one holding a
// module or a declared dependency that is not checked out at its recorded
// commit, or that itself has gitlinks, or whose tree tracks an aliasing
// symlink.
func submoduleAliases(root, head string, modules []validationModule, meta scopeMetadata) (string, error) {
	links, err := gitlinks(root, head)
	if err != nil {
		return "", err
	}
	holds := func(sub string) bool {
		for _, m := range modules {
			if m.Rel == sub || strings.HasPrefix(m.Rel, sub+"/") {
				return true
			}
		}
		for _, module := range meta.Modules {
			for _, dep := range module.DependsOn {
				dep = filepath.ToSlash(filepath.Clean(dep))
				if dep == "." || dep == sub || strings.HasPrefix(dep, sub+"/") || strings.HasPrefix(sub, dep+"/") {
					return true
				}
			}
		}
		return false
	}
	for sub, commit := range links {
		if !holds(sub) {
			continue
		}
		repo := filepath.Join(root, filepath.FromSlash(sub))
		if err := posemodel.RawGit("-C", repo, "cat-file", "-e", commit+"^{commit}").Run(); err != nil {
			return fmt.Sprintf("submodule %s is not checked out at its recorded commit, so aliases in it cannot be ruled out", sub), nil
		}
		nested, err := gitlinks(repo, commit)
		if err != nil {
			return "", err
		}
		if len(nested) > 0 {
			return fmt.Sprintf("submodule %s has nested submodules, so aliases in it cannot be ruled out", sub), nil
		}
		link, err := aliasingSymlinkIn(root, repo, commit, sub, modules, meta)
		if err != nil {
			return "", err
		}
		if link != "" {
			return fmt.Sprintf("the tree tracks symlink %s into or out of a module or a declared dependency, so changes cannot be attributed", link), nil
		}
	}
	return "", nil
}

// carryRelevantDirs are the directories whose case behaviour decides whether
// lexical selection is sound: every module directory and the nearest existing
// directory of every declared dependency.
func carryRelevantDirs(root string, modules []validationModule, meta scopeMetadata) []string {
	dirs := []string{}
	for _, m := range modules {
		dirs = append(dirs, m.Abs)
	}
	for _, module := range meta.Modules {
		for _, dep := range module.DependsOn {
			p := filepath.Join(root, filepath.FromSlash(filepath.ToSlash(filepath.Clean(dep))))
			for {
				if info, err := os.Stat(p); err == nil && info.IsDir() {
					dirs = append(dirs, p)
					break
				}
				parent := filepath.Dir(p)
				if parent == p || len(parent) < len(root) {
					break
				}
				p = parent
			}
		}
	}
	return dirs
}

// casefoldedDirectory reports whether dir looks names up without case: an
// entry spelled with its letters' case swapped names the same file.
func casefoldedDirectory(dir string, readDir func(string) ([]os.DirEntry, error), stat func(string) (os.FileInfo, error)) bool {
	entries, err := readDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		name := entry.Name()
		swapped := swapCase(name)
		if swapped == name {
			continue
		}
		original, err := stat(filepath.Join(dir, name))
		if err != nil {
			return false
		}
		other, err := stat(filepath.Join(dir, swapped))
		return err == nil && os.SameFile(original, other)
	}
	return false
}

func swapCase(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case unicode.IsUpper(r):
			return unicode.ToLower(r)
		case unicode.IsLower(r):
			return unicode.ToUpper(r)
		}
		return r
	}, s)
}
