package pose

import (
	"errors"
	"fmt"
	"sort"
)

// WithGovernanceWriteGuardForProjects runs a domain writer under the exact
// previously discovered project set. The callback receives a fresh Store with
// an operation-local resolver bound to those guarded roots; callers must
// recompute their material set under the guard and refuse any expansion.
// Foreign projects must have explicit RootsConfig bindings. Resolver read
// authorization and directory discovery do not grant write coordination.
func (s Store) WithGovernanceWriteGuardForProjects(operation string, projectIDs []string, callback func(Store) error) error {
	return s.withExplicitGovernanceWriteGuard(operation, projectIDs, callback)
}

// withExplicitGovernanceWriteGuard locks the local project and the exact set
// of foreign projects already discovered from the operation's material. A
// foreign root is eligible only when RootsConfig binds it explicitly; read
// discovery and ArtifactResolver.Authorize are not write coordination grants.
// The callback receives a fresh Store whose resolver is rebound to the guarded
// Stores and cannot resolve a project outside this set.
func (s Store) withExplicitGovernanceWriteGuard(operation string, projects []string, callback func(Store) error) error {
	if callback == nil {
		return governanceGuardError(ErrGovernanceWriteGuardInvalid, operation, "", errors.New("callback is required"))
	}
	localID := s.FederatedProjectID
	if localID == "" {
		var err error
		localID, err = s.WriteProjectID()
		if err != nil {
			return err
		}
	}
	if ValidateSlug(localID) != nil {
		return governanceGuardError(ErrGovernanceWriteGuardRootBinding, operation, "", errors.New("local project identity is unavailable"))
	}

	selected := map[string]bool{localID: true}
	for _, id := range projects {
		if ValidateSlug(id) != nil {
			return governanceGuardError(ErrGovernanceWriteGuardRootBinding, operation, "", errors.New("invalid project identity in material set"))
		}
		selected[id] = true
	}
	ids := make([]string, 0, len(selected))
	for id := range selected {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	resolver := ArtifactResolver{}
	if s.FederatedResolver != nil {
		resolver = *s.FederatedResolver
	} else if len(ids) > 1 {
		var err error
		resolver, _, err = EnvironmentArtifactResolverIn(s.fsys(), s.Root, "")
		if err != nil {
			return governanceGuardError(ErrGovernanceWriteGuardUnavailable, operation, "", err)
		}
	}
	if len(ids) > 1 && resolver.Roots == nil {
		return governanceGuardError(ErrGovernanceWriteGuardUnavailable, operation, "", errors.New("federated roots are unavailable"))
	}

	stores := make([]Store, 0, len(ids))
	canonical := make(map[string]string, len(ids))
	for _, id := range ids {
		store := s
		if id != localID {
			configured := resolver.Roots.cfg.Explicit[id]
			if configured == "" {
				return governanceGuardError(ErrGovernanceWriteGuardRootBinding, operation, id, errors.New("foreign writes require an explicit project root"))
			}
			var err error
			store, err = resolver.Roots.StoreFor(id)
			if err != nil {
				return governanceGuardError(ErrGovernanceWriteGuardUnavailable, operation, id, err)
			}
			want, err := canonicalGovernanceRoot(configured)
			if err != nil {
				return governanceGuardError(ErrGovernanceWriteGuardUnavailable, operation, id, err)
			}
			got, err := canonicalGovernanceRoot(store.Root)
			if err != nil || governanceRootKey(got) != governanceRootKey(want) {
				return governanceGuardError(ErrGovernanceWriteGuardRootBinding, operation, id, err)
			}
		}
		store.FederatedProjectID = id
		root, err := canonicalGovernanceRoot(store.Root)
		if err != nil {
			return governanceGuardError(ErrGovernanceWriteGuardUnavailable, operation, id, err)
		}
		canonical[id] = root
		stores = append(stores, store)
	}
	if lease, present := s.governanceWriteGuardInFiles(); present {
		if lease == nil || !lease.Active() {
			return governanceGuardError(ErrGovernanceWriteGuardReleased, operation, "", nil)
		}
		guardedByProject := make(map[string]Store, len(stores))
		for i, id := range ids {
			store := stores[i]
			if !lease.Covers(store) {
				return governanceGuardError(ErrGovernanceWriteGuardRootBinding, operation, id, errors.New("the carried lease does not cover every material root"))
			}
			childLease, ok := store.GovernanceWriteLease()
			if !ok || childLease != lease {
				return governanceGuardError(ErrGovernanceWriteGuardRootBinding, operation, id, errors.New("material store does not carry the same explicit lease"))
			}
			guardedByProject[id] = store
		}
		local, ok := guardedByProject[localID]
		if !ok {
			return governanceGuardError(ErrGovernanceWriteGuardRootBinding, operation, localID, errors.New("local project was not guarded"))
		}
		if resolver.Roots != nil {
			guardedResolver := resolver
			guardedResolver.Roots = resolver.Roots.withGovernanceStores(guardedByProject)
			guardedResolver.FS = newGovernanceRootVectorFiles(guardedByProject, localID)
			guardedResolver.snapshots = nil
			priorAuthorize := resolver.Authorize
			guardedResolver.Authorize = func(projectID string) bool {
				return selected[projectID] && (priorAuthorize == nil || priorAuthorize(projectID))
			}
			local.FederatedResolver = &guardedResolver
		}
		return callback(local.withoutGovernanceReadView())
	}

	return WithGovernanceWriteGuard(operation, stores, func(guarded []Store) error {
		guardedByProject := make(map[string]Store, len(guarded))
		for i, id := range ids {
			store := guarded[i]
			if root, err := canonicalGovernanceRoot(store.Root); err != nil || governanceRootKey(root) != governanceRootKey(canonical[id]) {
				return governanceGuardError(ErrGovernanceWriteGuardRootBinding, operation, id, err)
			}
			guardedByProject[id] = store
		}
		local, ok := guardedByProject[localID]
		if !ok {
			return governanceGuardError(ErrGovernanceWriteGuardRootBinding, operation, localID, errors.New("local project was not guarded"))
		}
		if resolver.Roots != nil {
			// Recheck the configured bindings after lock acquisition, then overlay
			// only the stores in this operation's exact set.
			for _, id := range ids {
				if id == localID {
					continue
				}
				if resolver.Roots.cfg.Explicit[id] == "" {
					return governanceGuardError(ErrGovernanceWriteGuardRootBinding, operation, id, errors.New("explicit project root disappeared"))
				}
				current, err := resolver.Roots.StoreFor(id)
				if err != nil {
					return governanceGuardError(ErrGovernanceWriteGuardUnavailable, operation, id, err)
				}
				root, err := canonicalGovernanceRoot(current.Root)
				if err != nil || governanceRootKey(root) != governanceRootKey(canonical[id]) {
					return governanceGuardError(ErrGovernanceWriteGuardRootBinding, operation, id, err)
				}
			}
			guardedResolver := resolver
			guardedResolver.Roots = resolver.Roots.withGovernanceStores(guardedByProject)
			guardedResolver.FS = newGovernanceRootVectorFiles(guardedByProject, localID)
			guardedResolver.snapshots = nil
			priorAuthorize := resolver.Authorize
			guardedResolver.Authorize = func(projectID string) bool {
				return selected[projectID] && (priorAuthorize == nil || priorAuthorize(projectID))
			}
			local.FederatedResolver = &guardedResolver
		}
		return callback(local)
	})
}

func governanceProjectSetEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for id := range a {
		if !b[id] {
			return false
		}
	}
	return true
}

func governanceProjectSetFromManifest(manifest *FederatedRoadmapManifest) map[string]bool {
	out := map[string]bool{}
	if manifest == nil {
		return out
	}
	for _, dep := range manifest.Dependencies {
		if dep.Identity.Project != "" {
			out[dep.Identity.Project] = true
		}
	}
	return out
}

func mergeGovernanceProjectSets(sets ...map[string]bool) map[string]bool {
	out := map[string]bool{}
	for _, set := range sets {
		for id := range set {
			out[id] = true
		}
	}
	return out
}

func governanceProjectIDs(set map[string]bool) []string {
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func requireSameGovernanceProjectSet(operation string, locked, observed map[string]bool) error {
	if governanceProjectSetEqual(locked, observed) {
		return nil
	}
	return governanceGuardError(ErrGovernanceWriteGuardRootBinding, operation, "", fmt.Errorf("material project set changed during guarded evaluation"))
}
