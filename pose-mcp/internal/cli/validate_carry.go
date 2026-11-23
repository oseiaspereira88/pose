package cli

// Carried checks (spec pose-validate-carries-unchanged-checks): with a
// comparable canonical result as base, validate --carry runs the modules the
// changed scope selects between the base head and HEAD and carries the base
// results of every other module, each stating the head it ran at. Any doubt
// about the base runs everything and records why (R2).

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"sort"
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
	if base.Cancelled {
		return unused("the canonical result was cancelled")
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
	// material_clean is the witness that the base validated its head. A spec
	// whose material the engine cannot compute is recorded unavailable in any
	// clean run, so an unavailable binding is not, by itself, a reason to
	// refuse; a run that was not clean has every binding unavailable and
	// material_clean false.
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
	// Module metadata is an administrative record, outside the cleanliness
	// check, so the selection policy must be the pinned head's (review round
	// 16, N46): the working copy must match it, and the one snapshot read from
	// the pinned commit is what every later step uses (review round 17, N50).
	if differs, err := workingFileDiffersFromHead(root, head, ".pose/indexes/module-metadata.json"); err != nil || differs {
		return unused("the working module metadata differs from HEAD's, so the selection policy is not the pinned one")
	}
	meta, err := scopeMetadataAt(root, head)
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
	// A declared dependency outside the repository changes without any change
	// the selection can see (review round 10, N28).
	for rel, module := range meta.Modules {
		for _, dep := range module.DependsOn {
			clean := filepath.ToSlash(filepath.Clean(dep))
			if filepath.IsAbs(dep) || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
				return unused("dependency %s of module %s lies outside the repository, so its changes cannot be seen", dep, rel)
			}
			// A path that is not already clean can traverse a symlink before
			// its ".." is applied (review round 11, N30).
			if filepath.ToSlash(dep) != clean {
				return unused("dependency %s of module %s is not a clean path, so its physical identity cannot be established", dep, rel)
			}
			// Non-ASCII names can alias under Unicode normalization (review
			// round 11, N32).
			if !asciiPath(clean) {
				return unused("dependency %s of module %s is not an ASCII path, so normalization aliases cannot be ruled out", dep, rel)
			}
			// Its physical identity, through any symlink tracked or ignored,
			// must be its spelling (review round 12, N33).
			if !physicallyLexical(root, clean) {
				return unused("dependency %s of module %s resolves through a symlink, so its changes cannot be attributed", dep, rel)
			}
		}
	}
	for _, m := range modules {
		if !asciiPath(m.Rel) {
			return unused("module %s is not an ASCII path, so normalization aliases cannot be ruled out", m.Rel)
		}
	}
	if sub := submodulesCheckedOut(root, head); sub != "" {
		return unused("submodule %s is not checked out at its recorded commit", sub)
	}
	// Index flags that hide tracked edits from status (review round 11, N31).
	if hidden, err := hiddenIndexEntries(root, head); err != nil {
		return unused("the index cannot be read: %v", err)
	} else if hidden != "" {
		return unused("%s is marked assume-unchanged or skip-worktree, so the working copy cannot be compared", hidden)
	}
	if caseInsensitiveFilesystem(root, os.Stat) {
		return unused("the filesystem is case-insensitive, so changes cannot be attributed to modules")
	}
	// A case-folded directory inside a case-sensitive filesystem is an alias
	// too (review round 8, N22).
	for _, dir := range carryRelevantDirs(root, modules, meta) {
		if casefoldedDirectory(dir, os.ReadDir, os.Lstat) {
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
	selection, err := computeChangedScopeWith(root, base.GitHead, head, modules, meta, carryExclusions(root, base.GitHead, head, resultsPath)...)
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

// submoduleAliases names a submodule that could hide an alias: one that is
// not checked out at its recorded commit, or that itself has gitlinks, or
// whose tree tracks an aliasing symlink.
func submoduleAliases(root, head string, modules []validationModule, meta scopeMetadata) (string, error) {
	links, err := gitlinks(root, head)
	if err != nil {
		return "", err
	}
	// Every gitlink is scanned, not only those holding a module or a
	// dependency: one inside a module aliases its paths too (review round 9,
	// N24).
	for sub, commit := range links {
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
// lexical selection is sound: every existing directory on the path from the
// root to each module and to each declared dependency, ancestors included,
// since any of them can fold a lookup (review round 9, N26).
func carryRelevantDirs(root string, modules []validationModule, meta scopeMetadata) []string {
	seen := map[string]bool{}
	dirs := []string{}
	add := func(p string) {
		root := filepath.Clean(root)
		for p = filepath.Clean(p); ; p = filepath.Dir(p) {
			if p != root && !strings.HasPrefix(p, root+string(filepath.Separator)) {
				return
			}
			if info, err := os.Stat(p); err == nil && info.IsDir() && !seen[p] {
				seen[p] = true
				dirs = append(dirs, p)
			}
			if p == root {
				return
			}
		}
	}
	for _, m := range modules {
		add(m.Abs)
	}
	for _, module := range meta.Modules {
		for _, dep := range module.DependsOn {
			add(filepath.Join(root, filepath.FromSlash(filepath.ToSlash(filepath.Clean(dep)))))
		}
	}
	sort.Strings(dirs)
	return dirs
}

// casefoldedDirectory reports whether dir looks names up without case: an
// entry spelled with its letters' case swapped names the same file.
func casefoldedDirectory(dir string, readDir func(string) ([]os.DirEntry, error), stat func(string) (os.FileInfo, error)) bool {
	// An unreadable directory cannot be shown not to fold. Entries are probed
	// without following links, and an unusable one is skipped, not taken as
	// proof (review round 10, N27).
	entries, err := readDir(dir)
	if err != nil {
		return true
	}
	// Only ASCII letters swap case without Unicode folding subtleties, such
	// as dotless i (review round 12, N35); a directory with no such entry
	// cannot be shown not to fold.
	for _, entry := range entries {
		name := entry.Name()
		if !asciiPath(name) || !strings.ContainsAny(name, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
			continue
		}
		swapped := swapCase(name)
		original, err := stat(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		// The first usable entry decides: its swapped spelling names the same
		// file only where lookups fold case.
		other, err := stat(filepath.Join(dir, swapped))
		return err == nil && os.SameFile(original, other)
	}
	return true
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

// asciiPath reports whether p is plain ASCII.
func asciiPath(p string) bool {
	for i := 0; i < len(p); i++ {
		if p[i] >= 0x80 {
			return false
		}
	}
	return true
}

// hiddenIndexEntries returns a tracked path, in the repository or in one of
// its checked-out submodules, whose index entry is marked assume-unchanged or
// skip-worktree: status cannot see edits to it.
func hiddenIndexEntries(root, head string) (string, error) {
	check := func(repo, prefix string) (string, error) {
		out, err := posemodel.RawGit("-C", repo, "ls-files", "-v", "-z").Output()
		if err != nil {
			return "", err
		}
		for _, entry := range strings.Split(string(out), "\x00") {
			if len(entry) < 3 {
				continue
			}
			tag := entry[0]
			if (tag >= 'a' && tag <= 'z') || tag == 'S' {
				return path.Join(prefix, entry[2:]), nil
			}
		}
		return "", nil
	}
	if hidden, err := check(root, ""); err != nil || hidden != "" {
		return hidden, err
	}
	links, err := gitlinks(root, head)
	if err != nil {
		return "", err
	}
	for sub := range links {
		repo := filepath.Join(root, filepath.FromSlash(sub))
		if _, err := os.Stat(filepath.Join(repo, ".git")); err != nil {
			continue
		}
		if hidden, err := check(repo, sub); err != nil || hidden != "" {
			return hidden, err
		}
	}
	return "", nil
}

// physicallyLexical reports whether rel, and every existing ancestor of it,
// resolves to its own spelling: no symlink, tracked or not, on the way.
func physicallyLexical(root, rel string) bool {
	root = filepath.Clean(root)
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return false
	}
	p := filepath.Join(root, filepath.FromSlash(rel))
	for ; p != root && strings.HasPrefix(p, root+string(filepath.Separator)); p = filepath.Dir(p) {
		if _, err := os.Lstat(p); err != nil {
			continue
		}
		resolved, err := filepath.EvalSymlinks(p)
		if err != nil {
			return false
		}
		want := filepath.Join(resolvedRoot, strings.TrimPrefix(p, root+string(filepath.Separator)))
		if filepath.Clean(resolved) != filepath.Clean(want) {
			return false
		}
	}
	return true
}

// headReflogLength counts the entries of HEAD's reflog; -1 when it cannot be
// read or reflogs are disabled.
func headReflogLength(root string) int {
	out, err := posemodel.RawGit("-C", root, "reflog", "show", "--format=%H", "HEAD", "--").Output()
	if err != nil {
		return -1
	}
	lines := strings.Count(string(out), "\n")
	if lines == 0 {
		return -1
	}
	return lines
}

// submodulesCheckedOut returns a gitlink of head whose working repository is
// missing or not at the recorded commit; "" when every one is.
func submodulesCheckedOut(root, head string) string {
	links, err := gitlinks(root, head)
	if err != nil {
		return "(gitlinks unreadable)"
	}
	for sub, commit := range links {
		repo := filepath.Join(root, filepath.FromSlash(sub))
		if _, err := os.Stat(filepath.Join(repo, ".git")); err != nil {
			return sub
		}
		out, err := posemodel.RawGit("-C", repo, "rev-parse", "HEAD").Output()
		if err != nil || strings.TrimSpace(string(out)) != commit {
			return sub
		}
	}
	return ""
}

// validationMaterialClean is the full predicate a run must satisfy at its
// start and at its end to validate head (review rounds 8–14): no tracked or
// untracked unignored difference in status (mode changes included), no index
// entry hiding edits, and every submodule checked out at its recorded commit
// (an uninitialized one reads as clean to status).
func validationMaterialClean(root, head string) bool {
	if !materialWorktreeClean(root) {
		return false
	}
	if hidden, err := hiddenIndexEntries(root, head); err != nil || hidden != "" {
		return false
	}
	if submodulesCheckedOut(root, head) != "" {
		return false
	}
	// Nested submodules are neither witnessed nor checked: a run over them
	// cannot claim its recorded material (review round 16, N48).
	links, err := gitlinks(root, head)
	if err != nil {
		return false
	}
	for sub, commit := range links {
		nested, err := gitlinks(filepath.Join(root, filepath.FromSlash(sub)), commit)
		if err != nil || len(nested) > 0 {
			return false
		}
	}
	return true
}

// finalizeCancellation records a cancelled run: not clean, and binding no
// material, whatever was decided before the signal was seen (review round 16,
// N47). It returns true for the run's cancelled field.
func finalizeCancellation(run *validationRunResult) bool {
	run.MaterialClean = false
	for spec := range run.MaterialProvenance {
		run.MaterialProvenance[spec] = posemodel.MaterialBindingUnavailable
	}
	return true
}

// workingFileDiffersFromHead reports whether rel in the working copy differs
// from its blob at head; absent on both sides is equal.
func workingFileDiffersFromHead(root, head, rel string) (bool, error) {
	present, err := inTree(root, head, rel)
	if err != nil {
		return true, err
	}
	working, werr := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if !present {
		// Absent from the commit: equal only when absent from the copy too.
		return !errors.Is(werr, os.ErrNotExist), nil
	}
	committed, cerr := posemodel.RawGit("-C", root, "show", head+":"+rel).Output()
	if cerr != nil {
		return true, fmt.Errorf("%s at %s is unreadable: %v", rel, head, cerr)
	}
	if werr != nil {
		return true, nil
	}
	return !bytes.Equal(working, committed), nil
}

// movementWitness summarizes the reflog lengths of HEAD and of every
// checked-out submodule's HEAD; "" when any cannot be read, so the witness
// never passes by default.
func movementWitness(root, head string) string {
	parent := headReflogLength(root)
	if parent < 0 || head == "" {
		return ""
	}
	parts := []string{fmt.Sprintf(".=%d", parent)}
	links, err := gitlinks(root, head)
	if err != nil {
		return ""
	}
	subs := make([]string, 0, len(links))
	for sub := range links {
		subs = append(subs, sub)
	}
	sort.Strings(subs)
	for _, sub := range subs {
		repo := filepath.Join(root, filepath.FromSlash(sub))
		if _, err := os.Stat(filepath.Join(repo, ".git")); err != nil {
			parts = append(parts, sub+"=absent")
			continue
		}
		n := headReflogLength(repo)
		if n < 0 {
			return ""
		}
		parts = append(parts, fmt.Sprintf("%s=%d", sub, n))
	}
	return strings.Join(parts, ",")
}

// scopeMetadataAt reads the module metadata of commit head: absent is empty,
// undecodable is an error.
func scopeMetadataAt(root, head string) (scopeMetadata, error) {
	var meta scopeMetadata
	const rel = ".pose/indexes/module-metadata.json"
	// Absence is read from the pinned tree; a listed blob that cannot be read
	// is an error, never an empty policy (review round 18, N52).
	present, err := inTree(root, head, rel)
	if err != nil {
		return meta, err
	}
	if !present {
		return meta, nil
	}
	raw, err := posemodel.RawGit("-C", root, "show", head+":"+rel).Output()
	if err != nil {
		return meta, fmt.Errorf("module metadata at %s is unreadable: %v", head, err)
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return scopeMetadata{}, fmt.Errorf("module metadata does not decode: %v", err)
	}
	return meta, nil
}

// inTree reports whether rel is an entry of head's tree, reading the tree,
// not the blob; a tree that cannot be read is an error.
func inTree(root, head, rel string) (bool, error) {
	out, err := posemodel.RawGit("-C", root, "ls-tree", "-z", head, "--", rel).Output()
	if err != nil {
		return false, fmt.Errorf("the tree of %s is unreadable: %v", head, err)
	}
	return strings.TrimSpace(strings.TrimRight(string(out), "\x00")) != "", nil
}
