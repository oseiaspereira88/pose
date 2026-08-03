package pose

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

const maxAssessmentFileSize = 4 << 20

type assessmentFile struct {
	RelPath   string
	AbsPath   string
	Component string
	Ext       string
	Content   string
}

func hasGeneratedAssessmentHeader(raw []byte) bool {
	prefix := strings.ToLower(string(raw))
	if len(prefix) > 512 {
		prefix = prefix[:512]
	}
	return strings.Contains(prefix, "code generated") && strings.Contains(prefix, "do not edit")
}

var skippedAssessmentDirs = map[string]bool{
	".git": true, ".pose": true, ".svn": true, ".hg": true,
	"node_modules": true, "vendor": true, "target": true, "dist": true,
	"build": true, "coverage": true, ".next": true, ".cache": true,
}

func shouldSkipAssessmentDir(name string) bool {
	return skippedAssessmentDirs[name] || strings.HasPrefix(name, ".")
}

func (s Store) projectLabel() string {
	abs, err := filepath.Abs(s.Root)
	if err != nil {
		return "project"
	}
	name := strings.TrimSpace(filepath.Base(filepath.Clean(abs)))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "project"
	}
	return name
}

func slugifyAssessmentPath(value string) string {
	value = filepath.ToSlash(filepath.Clean(value))
	if value == "." || value == "" {
		return "root"
	}
	var out strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			out.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(out.String(), "-")
	if slug != "" {
		return slug
	}
	digest := sha256.Sum256([]byte(value))
	return fmt.Sprintf("component-%x", digest[:6])
}

func (s Store) resolveComponentDir(relPath string) (string, string, error) {
	if strings.TrimSpace(relPath) == "" {
		relPath = "."
	}
	if filepath.IsAbs(relPath) {
		return "", "", fmt.Errorf("component path must be relative to the project root")
	}
	clean := filepath.Clean(relPath)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("component path escapes the project root")
	}

	rootAbs, err := filepath.Abs(s.Root)
	if err != nil {
		return "", "", fmt.Errorf("resolve project root: %w", err)
	}
	candidate := filepath.Join(rootAbs, clean)
	rel, err := filepath.Rel(rootAbs, candidate)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("component path escapes the project root")
	}

	rootReal, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", "", fmt.Errorf("resolve project root symlinks: %w", err)
	}
	candidateReal, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", "", fmt.Errorf("resolve component path: %w", err)
	}
	realRel, err := filepath.Rel(rootReal, candidateReal)
	if err != nil || realRel == ".." || strings.HasPrefix(realRel, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("component path resolves outside the project root")
	}
	info, err := os.Stat(candidateReal)
	if err != nil {
		return "", "", fmt.Errorf("stat component path: %w", err)
	}
	if !info.IsDir() {
		return "", "", fmt.Errorf("component path is not a directory")
	}
	return filepath.ToSlash(clean), candidateReal, nil
}

func isAssessmentTextExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".go", ".rs", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs",
		".py", ".java", ".kt", ".kts", ".cs", ".c", ".cc", ".cpp", ".h",
		".hpp", ".rb", ".php", ".swift", ".scala", ".sh", ".proto", ".graphql",
		".gql", ".json", ".yaml", ".yml", ".toml", ".xml":
		return true
	default:
		return false
	}
}

func isDebtSourceExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".go", ".rs", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs",
		".py", ".java", ".kt", ".kts", ".cs", ".c", ".cc", ".cpp", ".h",
		".hpp", ".rb", ".php", ".swift", ".scala", ".sh":
		return true
	default:
		return false
	}
}

func componentForRelativeFile(relFile string, roots []string) string {
	relFile = filepath.ToSlash(relFile)
	best := ""
	for _, root := range roots {
		root = strings.TrimSuffix(filepath.ToSlash(filepath.Clean(root)), "/")
		if root == "." {
			continue
		}
		if (relFile == root || strings.HasPrefix(relFile, root+"/")) && len(root) > len(best) {
			best = root
		}
	}
	if best != "" {
		return slugifyAssessmentPath(best)
	}
	first, _, _ := strings.Cut(relFile, "/")
	if first == "" || first == "." || first == filepath.Base(relFile) {
		return "root"
	}
	return slugifyAssessmentPath(first)
}

func (s Store) assessmentFiles() ([]assessmentFile, error) {
	roots := s.FindComponentDirectories()
	sort.SliceStable(roots, func(i, j int) bool { return len(roots[i]) > len(roots[j]) })

	seen := make(map[string]bool)
	var files []assessmentFile
	rootAbs, err := filepath.Abs(s.Root)
	if err != nil {
		return nil, err
	}
	rootReal, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return nil, err
	}
	_, absRoot, err := s.resolveComponentDir(".")
	if err != nil {
		return nil, err
	}
	err = filepath.WalkDir(absRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			if path != absRoot && shouldSkipAssessmentDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !isAssessmentTextExt(ext) {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.Size() > maxAssessmentFileSize {
			return nil
		}
		rel, err := filepath.Rel(rootReal, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if seen[rel] {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil || strings.IndexByte(string(raw), 0) >= 0 {
			return nil
		}
		if hasGeneratedAssessmentHeader(raw) {
			return nil
		}
		seen[rel] = true
		files = append(files, assessmentFile{
			RelPath: rel, AbsPath: path, Component: componentForRelativeFile(rel, roots),
			Ext: ext, Content: string(raw),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].RelPath < files[j].RelPath })
	return files, nil
}
