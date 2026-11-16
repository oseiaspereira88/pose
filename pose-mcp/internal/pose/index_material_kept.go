package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// keptIndexMaterial holds, per process, the digest of the federated input
// material of an index for one state of every authorized repository. Every
// review operation checks that the index it reads is current, and that check
// recomputed the federated material (about 90 s of a 140 s closeout on
// harne8 with pose-dist as a dependency), so a long-running MCP server spent
// minutes on every call (spec pose-index-material-is-kept-per-repository-state).
var keptIndexMaterial = newKeptCache[string](256)

// indexMaterialCompute, when set, replaces the build of the material; tests
// use it to count calls.
var indexMaterialCompute func(Store) (IndexFederatedMaterial, error)

// indexMaterialStateLimit bounds the dirty and untracked paths hashed for a
// key; past it the material is computed and not kept.
const indexMaterialStateLimit = 4096

// keptIndexMaterialDigest returns the federated material digest of store,
// kept for the repository state it was computed from.
func keptIndexMaterialDigest(store Store) (string, bool) {
	compute := func() (string, bool) {
		build := BuildIndexFederatedMaterialIn
		if indexMaterialCompute != nil {
			build = indexMaterialCompute
		}
		material, err := build(store)
		if err != nil {
			return "", false
		}
		return IndexFederatedMaterialDigest(material.Projection)
	}
	key, ok := indexMaterialStateKey(store)
	if !ok {
		return compute()
	}
	kept := keptIndexMaterial.get(key, func() (string, bool) {
		digest, known := compute()
		return digest, known
	})
	return kept, kept != ""
}

// indexMaterialStateKey names the state every input of the material comes
// from: the engine, the project, and for the operation's root and every
// authorized root, its HEAD, its full status (tracked, untracked, submodule
// entries) and the bytes of each path the status lists. A file changed twice
// shows the same status line, so its bytes are part of the key. Without an
// explicit resolver, or past the bound, there is no key.
func indexMaterialStateKey(store Store) (string, bool) {
	if store.FederatedResolver == nil || store.FederatedResolver.Roots == nil {
		return "", false
	}
	roots := map[string]string{"": store.Root}
	for _, project := range store.FederatedResolver.Roots.Projects() {
		source, err := store.FederatedResolver.Roots.StoreFor(project)
		if err != nil {
			return "", false
		}
		roots[project] = source.Root
	}
	names := make([]string, 0, len(roots))
	for name := range roots {
		names = append(names, name)
	}
	sort.Strings(names)
	hash := sha256.New()
	hash.Write([]byte(engineVersion + "\x00" + store.FederatedProjectID + "\x00"))
	hashed := 0
	for _, name := range names {
		root, err := filepath.Abs(roots[name])
		if err != nil {
			return "", false
		}
		state, count, ok := repositoryState(root)
		if !ok {
			return "", false
		}
		hashed += count
		if hashed > indexMaterialStateLimit {
			return "", false
		}
		hash.Write([]byte(name + "\x00" + root + "\x00" + state + "\x00"))
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), true
}

// repositoryState digests a repository's HEAD, its status and the bytes of
// the paths its status lists, and counts those paths.
func repositoryState(root string) (string, int, bool) {
	// Without optional locks, so status does not refresh and rewrite the
	// index file the operation's read view observes.
	env := []string{"GIT_OPTIONAL_LOCKS=0", "LC_ALL=C", "LANGUAGE=C"}
	head, err := gitOutputIn(root, env, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return "", 0, false
	}
	status, err := gitOutputIn(root, env, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--ignore-submodules=none")
	if err != nil {
		return "", 0, false
	}
	hash := sha256.New()
	hash.Write([]byte(strings.TrimSpace(head) + "\x00" + status + "\x00"))
	count := 0
	records := strings.Split(status, "\x00")
	for i := 0; i < len(records); i++ {
		record := records[i]
		if len(record) < 4 {
			continue
		}
		path := record[3:]
		if record[0] == 'R' || record[0] == 'C' {
			i++ // the source path of a rename or copy follows
		}
		count++
		full := filepath.Join(root, filepath.FromSlash(path))
		info, err := os.Lstat(full)
		switch {
		case err != nil:
			hash.Write([]byte(path + "\x00missing\x00"))
		case info.IsDir():
			hash.Write([]byte(path + "\x00dir\x00"))
		case info.Mode()&os.ModeSymlink != 0:
			target, _ := os.Readlink(full)
			hash.Write([]byte(path + "\x00link\x00" + target + "\x00"))
		default:
			raw, err := os.ReadFile(full)
			if err != nil {
				return "", 0, false
			}
			sum := sha256.Sum256(raw)
			hash.Write([]byte(path + "\x00" + hex.EncodeToString(sum[:]) + "\x00"))
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), count, true
}
