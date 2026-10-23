package pose

// releaseManifestFixture is a manifest as `pose release prepare` writes it: every redundant field
// follows from the fragments, so it is valid under the contract of
// pose-release-manifest-tag-integrity.
func releaseManifestFixture(version string, fragments []ReleaseFragment) string {
	if fragments == nil {
		fragments = []ReleaseFragment{}
	}
	for i := range fragments {
		if fragments[i].Category == "" {
			fragments[i].Category = "fixed"
		}
	}
	manifest := NewReleaseManifest(version, "", "2026-10-02T00:00:00Z", fragments, ReleasePolicy{}, ChangelogPolicy{}, map[string]string{"source": "VERSION", "kind": "text", "value": version})
	return string(CanonicalJSON(manifest))
}
