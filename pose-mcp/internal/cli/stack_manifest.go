package cli

// Shared manifest-to-stack classification (spec
// pose-validation-scanner-consolidation): `discoverValidationModules`
// (validate.go, used by pose validate/install/init) and `scanModules`
// (index.go, used by pose index) each walked the tree with their own
// independent filename-to-stack switch. A stack-detection fix applied to
// one silently did not apply to the other — this function is the single
// place that mapping lives now.

import posepkg "github.com/harne8/pose-mcp/internal/pose"

// stackForManifestFile returns the stack a manifest/marker file name
// signals, or "" when the name is not recognized. The Java/Python
// alternatives (build.gradle vs pom.xml; poetry/pipenv/pip/setuptools/
// pep517) all resolve to their one language here — which manager applies
// is a validation-matrix.json `when` concern, not a discovery concern (spec
// pose-stack-catalog-expansion).
func stackForManifestFile(name string) string {
	return posepkg.StackForManifestFile(name)
}
