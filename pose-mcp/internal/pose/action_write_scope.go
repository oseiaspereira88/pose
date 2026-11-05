package pose

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// actionResolutionProjectID prefers the Store's selected project, but only
// while its declared identity and any resolver binding still name this root.
// A derived directory/environment identity is used only for an unscoped Store.
func (s Store) actionResolutionProjectID() (string, error) {
	project := strings.TrimSpace(s.FederatedProjectID)
	if project == "" {
		return s.WriteProjectID()
	}
	if ValidateSlug(project) != nil {
		return "", fmt.Errorf("pose: selected project identity is invalid")
	}
	declared, found, err := ReadProjectFileIn(s.fsys(), s.Root)
	if err != nil {
		return "", err
	}
	if found && declared != project {
		return "", fmt.Errorf("pose: selected project identity conflicts with .pose/project.json")
	}
	if s.FederatedResolver != nil {
		resolver := s.FederatedResolver
		if resolver.Authorize != nil && !resolver.Authorize(project) {
			return "", fmt.Errorf("pose: selected project is not authorized")
		}
		if resolver.Roots != nil {
			selected, selectErr := resolver.Roots.StoreFor(project)
			if selectErr == nil {
				if !sameProjectRoot(s.fsys(), s.Root, selected.Root) {
					return "", fmt.Errorf("pose: selected project is bound to another root")
				}
				if resolver.Roots.cfg.DefaultProjectID != "" && resolver.Roots.cfg.DefaultProjectID != project &&
					resolver.Roots.cfg.DefaultRoot != "" &&
					sameProjectRoot(s.fsys(), s.Root, resolver.Roots.cfg.DefaultRoot) {
					return "", fmt.Errorf("pose: selected project conflicts with the default identity for this root")
				}
			} else if isUnknownProjectSelection(selectErr) {
				// An explicitly selected Store can be the local default root even
				// when Roots has no nonempty project-id entry for it. Confirm the
				// default root and its declared identity before using that fallback.
				defaultStore, defaultErr := resolver.Roots.StoreFor("")
				if defaultErr != nil || !sameProjectRoot(s.fsys(), s.Root, defaultStore.Root) ||
					resolver.Roots.cfg.DefaultProjectID != "" && resolver.Roots.cfg.DefaultProjectID != project {
					return "", fmt.Errorf("pose: selected project is not bound to this root")
				}
			} else {
				return "", selectErr
			}
		}
	}
	return project, nil
}

// actionMaterialProjectSet identifies the roots a request evaluation actually
// resolves. It follows each request reference with the same resolver used by
// actionMaterial, retaining original, canonical and transitive dependency
// projects so redirects cannot move the evaluated subject outside the lease.
func (s Store) actionMaterialProjectSet(request ActionRequest) (map[string]bool, error) {
	localID := request.Project
	if localID == "" {
		var err error
		localID, err = s.WriteProjectID()
		if err != nil {
			return nil, err
		}
	}
	resolver := ArtifactResolver{}
	if s.FederatedResolver != nil {
		resolver = *s.FederatedResolver
	} else {
		var err error
		resolver, _, err = EnvironmentArtifactResolverIn(s.fsys(), s.Root, "")
		if err != nil {
			return nil, err
		}
	}
	probe := s
	probe.FederatedProjectID = localID
	probe.FederatedResolver = &resolver
	projects := map[string]bool{localID: true}
	addResolution := func(raw string) error {
		ref, err := ParseArtifactRef(raw)
		if err != nil {
			return err
		}
		if ref.Project == "" {
			ref.Project = localID
		}
		if ref.Project == localID {
			// Match actionMaterial: an already-selected local Store does not
			// need federation adoption or a contained foreign-root resolver.
			// addNode still validates the actual artifact/node in that Store.
			return nil
		}
		resolved := resolver.Resolve(localID, ref.String())
		if !resolved.Resolved {
			return fmt.Errorf("material %s: %s", raw, resolved.State)
		}
		addArtifactProject(projects, resolved.Identity)
		if resolved.CanonicalIdentity != nil {
			addArtifactProject(projects, *resolved.CanonicalIdentity)
		}
		for _, dependency := range resolved.Dependencies {
			addArtifactProject(projects, dependency)
		}
		return nil
	}
	var addNode func(NodeRef) error
	addNode = func(node NodeRef) error {
		ref, err := ParseArtifactRef(node.Artifact)
		if err != nil {
			return err
		}
		if ref.Project == "" {
			ref.Project = localID
		}
		if err := addResolution(ref.String()); err != nil {
			return err
		}
		source, materialRef, _, err := probe.actionMaterial(localID, node)
		if err != nil {
			return err
		}
		addArtifactProject(projects, materialRef)
		if source.FederatedProjectID != "" {
			projects[source.FederatedProjectID] = true
		}
		if materialRef.Kind == "roadmap" || materialRef.Kind == "milestone" {
			// Membership can be partly known: retain every resolvable member in
			// the lease set while leaving unresolved foreign members unknown.
			// Consumers report that coverage limitation and keep known local
			// restrictions; the writer never invents a root for the unknown member.
			members, _ := probe.actionDestinations(localID, node)
			for _, member := range members {
				memberRef, parseErr := ParseArtifactRef(member.Artifact)
				if parseErr != nil {
					return parseErr
				}
				if memberRef.Project == "" {
					memberRef.Project = localID
				}
				if err := addResolution(memberRef.String()); err != nil {
					if memberRef.Project != localID {
						continue
					}
					return err
				}
				memberStore, canonical, _, materialErr := probe.actionMaterial(localID, member)
				if materialErr != nil {
					if memberRef.Project != localID {
						continue
					}
					return materialErr
				}
				addArtifactProject(projects, canonical)
				if memberStore.FederatedProjectID != "" {
					projects[memberStore.FederatedProjectID] = true
				}
			}
		}
		return nil
	}
	for _, target := range request.Targets {
		if err := addNode(target); err != nil {
			return nil, err
		}
	}
	for _, effect := range request.Effects {
		for _, scope := range effect.Scope {
			if err := addNode(scope); err != nil {
				return nil, err
			}
		}
	}
	if request.Subject != nil && request.Subject.Node.Artifact != "" {
		if err := addNode(request.Subject.Node); err != nil {
			return nil, err
		}
	}
	if request.Kind == ActionDelegatedReview && request.Review != nil {
		bundle, err := probe.PrepareReviewBundle(request.Review.Scope)
		if err != nil {
			return nil, err
		}
		for id := range governanceProjectSetFromManifest(bundle.Payload.FederatedManifest) {
			projects[id] = true
		}
	}
	return projects, nil
}

func addArtifactProject(projects map[string]bool, ref ArtifactRef) {
	if ref.Project != "" {
		projects[ref.Project] = true
	}
}

func (s Store) readActionRequestEventsForWrite(id string) ([]ActionEvent, error) {
	if err := s.checkActionJournal(id); err != nil {
		return nil, err
	}
	raw, err := s.fsys().ReadFile(actionJournalPath(s.Root, id))
	if err != nil {
		return nil, err
	}
	events, err := parseActionJournal(raw)
	if err != nil {
		return nil, fmt.Errorf("pose: action request %s: %w", id, err)
	}
	if len(events) == 0 || events[0].Type != ActionEventOpened || events[0].Request == nil || events[0].Request.ID != id {
		return nil, errors.New("pose: action journal does not start with its opening")
	}
	if ActionRequestDigest(*events[0].Request) != events[0].Request.RequestDigest {
		return nil, errors.New("pose: action request digest does not match its content")
	}
	if err := s.verifyActionAnswers(events); err != nil {
		return nil, err
	}
	return events, nil
}

func (s Store) actionRequestsForRecovery(id string, request ActionRequest) ([]ActionRequest, error) {
	requests := []ActionRequest{request}
	intentIDs := []string{id}
	if request.Supersedes != "" {
		intentIDs = append(intentIDs, request.Supersedes)
		prior, err := s.readActionRequestEventsForWrite(request.Supersedes)
		if err != nil {
			return nil, err
		}
		requests = append(requests, *prior[0].Request)
	}
	for _, intentID := range uniqueSorted(intentIDs) {
		intent, err := s.readActionIntent(intentID)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if intent != nil {
			requests = append(requests, intent.Successor)
		}
	}
	return requests, nil
}

func (s Store) actionRequestsProjectSet(requests []ActionRequest) (map[string]bool, error) {
	sets := make([]map[string]bool, 0, len(requests))
	for _, request := range requests {
		set, err := s.actionMaterialProjectSet(request)
		if err != nil {
			return nil, err
		}
		sets = append(sets, set)
	}
	return mergeGovernanceProjectSets(sets...), nil
}
