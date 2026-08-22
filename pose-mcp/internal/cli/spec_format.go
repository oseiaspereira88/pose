package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

var datePrefixRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-`)

type SpecMigrationItem struct {
	Slug         string `json:"slug"`
	SourcePath   string `json:"source_path"`
	TargetPath   string `json:"target_path"`
	Format       string `json:"format"` // "folder" or "flat"
	HasCompanion bool   `json:"has_companion"`
	Status       string `json:"status"` // "migrated", "skipped", "dry-run", "error"
	Error        string `json:"error,omitempty"`
}

// cmdSpecFormat handles the `pose spec-format` command suite.
func cmdSpecFormat(root string, args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	if len(args) == 0 {
		fmt.Fprintln(stderr, cliText(locale,
			"Usage: pose spec-format <migrate|status> [<slug>|--all] [--format folder|flat] [--dry-run] [--json]",
			"Uso: pose spec-format <migrate|status> [<slug>|--all] [--format folder|flat] [--dry-run] [--json]"))
		return 2
	}

	subcmd := args[0]
	subargs := args[1:]

	switch subcmd {
	case "migrate":
		return cmdSpecFormatMigrate(root, subargs, stdout, stderr, locale)
	case "status":
		return cmdSpecFormatStatus(root, subargs, stdout, stderr, locale)
	case "-h", "--help":
		dispatchCommandHelp("spec-format", nil, stdout, locale)
		return 0
	default:
		fmt.Fprintf(stderr, cliText(locale, "pose spec-format: unknown subcommand %q\n", "pose spec-format: subcomando desconhecido %q\n"), subcmd)
		return 2
	}
}

func cmdSpecFormatMigrate(root string, args []string, stdout, stderr io.Writer, locale cliLocale) int {
	targetSlug := ""
	allSpecs := false
	formatPref := "folder" // default preference
	dryRun := false
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			allSpecs = true
		case "--dry-run":
			dryRun = true
		case "--json":
			jsonOutput = true
		case "--format":
			if i+1 < len(args) {
				formatPref = args[i+1]
				i++
			}
		case "-h", "--help":
			dispatchCommandHelp("spec-format", []string{"migrate"}, stdout, locale)
			return 0
		default:
			if !strings.HasPrefix(args[i], "-") && targetSlug == "" {
				targetSlug = args[i]
			}
		}
	}

	if targetSlug == "" && !allSpecs {
		fmt.Fprintln(stderr, cliText(locale,
			"Usage: pose spec-format migrate <slug>|--all [--format folder|flat] [--dry-run] [--json]",
			"Uso: pose spec-format migrate <slug>|--all [--format folder|flat] [--dry-run] [--json]"))
		return 2
	}

	store := pose.Store{Root: root}
	specsDir := filepath.Join(root, ".pose", "specs")
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		fmt.Fprintf(stderr, "pose spec-format: reading specs dir: %v\n", err)
		return 1
	}

	results := []SpecMigrationItem{}

	for _, e := range entries {
		name := e.Name()
		if strings.EqualFold(name, "README.md") || strings.EqualFold(name, ".gitkeep") {
			continue
		}

		var specPath string
		var isDir bool
		if e.IsDir() {
			cand := filepath.Join(specsDir, name, "spec.md")
			if _, err := os.Stat(cand); err == nil {
				specPath = cand
				isDir = true
			}
		} else if strings.HasSuffix(name, ".md") {
			specPath = filepath.Join(specsDir, name)
			isDir = false
		}

		if specPath == "" {
			continue
		}

		sp, err := store.GetSpec(name)
		if err != nil {
			// Try parsing directly
			raw, err := os.ReadFile(specPath)
			if err != nil {
				continue
			}
			fm, _ := pose.SplitFrontmatter(string(raw))
			slug := fm["slug"]
			if slug == "" {
				slug = strings.TrimSuffix(name, ".md")
			}
			sp = &pose.Spec{
				Slug:        slug,
				CreatedAt:   fm["created_at"],
				CompletedAt: fm["completed_at"],
				Path:        specPath,
			}
		}

		if targetSlug != "" && sp.Slug != targetSlug && name != targetSlug {
			continue
		}

		// Check companion files in source directory
		hasCompanion := false
		sourceDir := ""
		if isDir {
			sourceDir = filepath.Join(specsDir, name)
			if dirEntries, err := os.ReadDir(sourceDir); err == nil {
				for _, de := range dirEntries {
					if de.Name() != "spec.md" && !strings.HasPrefix(de.Name(), ".") {
						hasCompanion = true
						break
					}
				}
			}
		}

		// Invariant: companion artifacts FORCE folder envelope
		targetFormat := formatPref
		if hasCompanion {
			targetFormat = "folder"
		}

		// Check if already in target format
		if datePrefixRegex.MatchString(name) {
			if targetFormat == "flat" {
				if !isDir {
					results = append(results, SpecMigrationItem{
						Slug:       sp.Slug,
						SourcePath: specPath,
						TargetPath: specPath,
						Format:     "flat",
						Status:     "skipped (already flat)",
					})
					continue
				}
				if hasCompanion {
					results = append(results, SpecMigrationItem{
						Slug:         sp.Slug,
						SourcePath:   specPath,
						TargetPath:   specPath,
						Format:       "folder",
						HasCompanion: true,
						Status:       "skipped (has companion/amends, kept as folder)",
					})
					continue
				}
				// isDir is true and hasCompanion is false: migrate to flat file
			} else if targetFormat == "folder" {
				if isDir {
					results = append(results, SpecMigrationItem{
						Slug:         sp.Slug,
						SourcePath:   specPath,
						TargetPath:   specPath,
						Format:       "folder",
						HasCompanion: hasCompanion,
						Status:       "skipped (already folder)",
					})
					continue
				}
				// isDir is false: migrate to folder
			} else {
				// Default format (auto): keep existing date-prefixed as-is
				results = append(results, SpecMigrationItem{
					Slug:         sp.Slug,
					SourcePath:   specPath,
					TargetPath:   specPath,
					Format:       formatPref,
					HasCompanion: hasCompanion,
					Status:       "skipped (already date-prefixed)",
				})
				continue
			}
		}

		// Determine date prefix
		dateStr := strings.TrimSpace(sp.CreatedAt)
		if datePrefixRegex.MatchString(name) {
			dateStr = name[:10]
		} else if len(dateStr) >= 10 && datePrefixRegex.MatchString(dateStr[:10]+"-") {
			dateStr = dateStr[:10]
		} else if len(sp.CompletedAt) >= 10 && datePrefixRegex.MatchString(sp.CompletedAt[:10]+"-") {
			dateStr = sp.CompletedAt[:10]
		} else {
			dateStr = time.Now().UTC().Format("2006-01-02")
		}

		var targetPath string
		var targetDir string
		if targetFormat == "folder" {
			targetDir = filepath.Join(specsDir, dateStr+"-"+sp.Slug)
			targetPath = filepath.Join(targetDir, "spec.md")
		} else {
			targetPath = filepath.Join(specsDir, dateStr+"-"+sp.Slug+".md")
		}

		item := SpecMigrationItem{
			Slug:         sp.Slug,
			SourcePath:   specPath,
			TargetPath:   targetPath,
			Format:       targetFormat,
			HasCompanion: hasCompanion,
			Status:       "migrated",
		}

		if dryRun {
			item.Status = "dry-run"
			results = append(results, item)
			continue
		}

		// Perform migration
		if targetFormat == "folder" {
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				item.Status = "error"
				item.Error = err.Error()
				results = append(results, item)
				continue
			}
			if isDir {
				// Move all files from sourceDir to targetDir
				if sourceDir != targetDir {
					if dirEntries, err := os.ReadDir(sourceDir); err == nil {
						for _, de := range dirEntries {
							oldFile := filepath.Join(sourceDir, de.Name())
							newFile := filepath.Join(targetDir, de.Name())
							_ = os.Rename(oldFile, newFile)
						}
					}
					_ = os.Remove(sourceDir)
				}
			} else {
				// Move single flat file into targetDir/spec.md
				_ = os.Rename(specPath, targetPath)
			}
		} else {
			// Target is flat file
			if isDir {
				// Move spec.md to flat file and remove directory
				_ = os.Rename(specPath, targetPath)
				_ = os.Remove(sourceDir)
			} else {
				_ = os.Rename(specPath, targetPath)
			}
		}

		results = append(results, item)
	}

	if jsonOutput {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(results)
		return 0
	}

	text := func(en, pt string) string { return cliText(locale, en, pt) }
	fmt.Fprintf(stdout, "%s", fmt.Sprintf(text("POSE Spec Format Migration (%d processed):\n\n", "Migração de Formato de Specs POSE (%d processadas):\n\n"), len(results)))

	for _, item := range results {
		relSource, _ := filepath.Rel(root, item.SourcePath)
		relTarget, _ := filepath.Rel(root, item.TargetPath)
		if item.Status == "dry-run" {
			fmt.Fprintf(stdout, "  [DRY-RUN] %-30s -> %s (format: %s, companion: %v)\n", item.Slug, relTarget, item.Format, item.HasCompanion)
		} else if strings.HasPrefix(item.Status, "skipped") {
			fmt.Fprintf(stdout, "  [SKIPPED] %-30s (%s)\n", item.Slug, relSource)
		} else if item.Status == "migrated" {
			fmt.Fprintf(stdout, "  [OK]      %-30s %s -> %s (format: %s)\n", item.Slug, relSource, relTarget, item.Format)
		} else {
			fmt.Fprintf(stdout, "  [ERROR]   %-30s %s (%v)\n", item.Slug, relSource, item.Error)
		}
	}
	fmt.Fprintln(stdout)
	return 0
}

func cmdSpecFormatStatus(root string, args []string, stdout, stderr io.Writer, locale cliLocale) int {
	jsonOutput := false
	for _, a := range args {
		if a == "--json" {
			jsonOutput = true
		} else if a == "-h" || a == "--help" {
			dispatchCommandHelp("spec-format", []string{"status"}, stdout, locale)
			return 0
		}
	}

	specsDir := filepath.Join(root, ".pose", "specs")
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		fmt.Fprintf(stderr, "pose spec-format: reading specs dir: %v\n", err)
		return 1
	}

	total := 0
	datePrefixed := 0
	legacy := 0

	for _, e := range entries {
		name := e.Name()
		if strings.EqualFold(name, "README.md") || strings.EqualFold(name, ".gitkeep") {
			continue
		}
		total++
		if datePrefixRegex.MatchString(name) {
			datePrefixed++
		} else {
			legacy++
		}
	}

	if jsonOutput {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]any{
			"total":         total,
			"date_prefixed": datePrefixed,
			"legacy":        legacy,
			"conforming":    legacy == 0,
		})
		return 0
	}

	text := func(en, pt string) string { return cliText(locale, en, pt) }
	fmt.Fprintf(stdout, "%s", text("POSE Spec Format Status:\n", "Status de Formato de Specs POSE:\n"))
	fmt.Fprintf(stdout, "  %s: %d\n", text("Total Specs", "Total de Specs"), total)
	fmt.Fprintf(stdout, "  %s: %d\n", text("Date-prefixed (Modern)", "Com prefixo de data (Moderno)"), datePrefixed)
	fmt.Fprintf(stdout, "  %s: %d\n\n", text("Legacy format", "Formato legado"), legacy)

	if legacy > 0 {
		fmt.Fprintf(stdout, "%s", text("Run `pose spec-format migrate --all` to modernize all specs.\n\n", "Execute `pose spec-format migrate --all` para modernizar todas as specs.\n\n"))
	} else {
		fmt.Fprintf(stdout, "%s", text("All specifications conform to the modern chronological format.\n\n", "Todas as especificações estão no formato cronológico moderno.\n\n"))
	}
	return 0
}
