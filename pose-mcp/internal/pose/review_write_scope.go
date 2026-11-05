package pose

import (
	"errors"
	"time"
)

// WithCloseoutWriteGuard runs a lifecycle close under the exact roots consumed
// by its current closeout state and review bundle. The guarded callback gets a
// freshly rebound Store and must use it for the final gates and publication.
func (s Store) WithCloseoutWriteGuard(operation, scope string, callback func(Store) error) error {
	return s.withCloseoutWriteGuard(operation, scope, callback)
}

func (s Store) closeoutWriteProjectSet(scope string) (map[string]bool, error) {
	localID := s.FederatedProjectID
	if localID == "" {
		var err error
		localID, err = s.WriteProjectID()
		if err != nil {
			return nil, err
		}
	}
	projects := map[string]bool{localID: true}
	state, err := s.GetCloseoutState(scope)
	if err != nil {
		return nil, err
	}
	if state.ReadView != nil {
		for _, binding := range state.ReadView.Bindings {
			if binding.ProjectID != "" {
				projects[binding.ProjectID] = true
			}
		}
	}
	if state.FederatedAcceptance != nil {
		projects = mergeGovernanceProjectSets(projects, governanceProjectSetFromManifest(&state.FederatedAcceptance.Manifest))
	}
	_, bundleProjects, err := s.reviewBundleProjectSet(scope)
	if err != nil {
		return nil, err
	}
	return mergeGovernanceProjectSets(projects, bundleProjects), nil
}

func (s Store) withCloseoutWriteGuard(operation, scope string, callback func(Store) error) error {
	if callback == nil {
		return governanceGuardError(ErrGovernanceWriteGuardInvalid, operation, "", errors.New("callback is required"))
	}
	lockedProjects, err := s.closeoutWriteProjectSet(scope)
	if err != nil {
		return err
	}
	return s.withExplicitGovernanceWriteGuard(operation, governanceProjectIDs(lockedProjects), func(guarded Store) error {
		observedProjects, err := guarded.closeoutWriteProjectSet(scope)
		if err != nil {
			return err
		}
		if err := requireSameGovernanceProjectSet(operation, lockedProjects, observedProjects); err != nil {
			return err
		}
		return callback(guarded)
	})
}

func (s Store) reviewBundleProjectSet(scope string) (ReviewBundle, map[string]bool, error) {
	bundle, err := s.PrepareReviewBundle(scope)
	if err != nil {
		return ReviewBundle{}, nil, err
	}
	projects := governanceProjectSetFromManifest(bundle.Payload.FederatedManifest)
	localID := s.FederatedProjectID
	if localID == "" {
		localID, err = s.WriteProjectID()
		if err != nil {
			return ReviewBundle{}, nil, err
		}
	}
	projects[localID] = true
	return bundle, projects, nil
}

func (s Store) withReviewBundleWriteGuard(operation, scope string, callback func(Store, ReviewBundle) error) error {
	if callback == nil {
		return governanceGuardError(ErrGovernanceWriteGuardInvalid, operation, "", errors.New("callback is required"))
	}
	_, lockedProjects, err := s.reviewBundleProjectSet(scope)
	if err != nil {
		return err
	}
	return s.withExplicitGovernanceWriteGuard(operation, governanceProjectIDs(lockedProjects), func(guarded Store) error {
		fresh, observedProjects, err := guarded.reviewBundleProjectSet(scope)
		if err != nil {
			return err
		}
		if err := requireSameGovernanceProjectSet(operation, lockedProjects, observedProjects); err != nil {
			return err
		}
		return callback(guarded, fresh)
	})
}

func (s Store) reviewAttestationProjectSet(att ReviewAttestation) (map[string]bool, error) {
	bundle, err := s.LoadReviewBundle(att.BundleID)
	if err != nil {
		return nil, err
	}
	localID := s.FederatedProjectID
	if localID == "" {
		localID, err = s.WriteProjectID()
		if err != nil {
			return nil, err
		}
	}
	projects := map[string]bool{localID: true}
	if reviewDecisionApproves(att.Decision) {
		_, current, err := s.reviewBundleProjectSet(bundle.Payload.Scope.Ref)
		if err != nil {
			return nil, err
		}
		projects = mergeGovernanceProjectSets(projects, current)
	}
	return projects, nil
}

func (s Store) withReviewAttestationWriteGuard(operation string, att ReviewAttestation, callback func(Store) error) error {
	if callback == nil {
		return governanceGuardError(ErrGovernanceWriteGuardInvalid, operation, "", errors.New("callback is required"))
	}
	lockedProjects, err := s.reviewAttestationProjectSet(att)
	if err != nil {
		return err
	}
	return s.withExplicitGovernanceWriteGuard(operation, governanceProjectIDs(lockedProjects), func(guarded Store) error {
		bundle, err := guarded.LoadReviewBundle(att.BundleID)
		if err != nil {
			return err
		}
		observed := map[string]bool{}
		localID := guarded.FederatedProjectID
		if localID == "" {
			localID, err = guarded.WriteProjectID()
			if err != nil {
				return err
			}
		}
		observed[localID] = true
		if reviewDecisionApproves(att.Decision) {
			_, current, prepErr := guarded.reviewBundleProjectSet(bundle.Payload.Scope.Ref)
			if prepErr != nil {
				return prepErr
			}
			observed = mergeGovernanceProjectSets(observed, current)
		}
		if err := requireSameGovernanceProjectSet(operation, lockedProjects, observed); err != nil {
			return err
		}
		return callback(guarded)
	})
}

func (s Store) reviewRunProjectSet(runID string) (map[string]bool, error) {
	localID := s.FederatedProjectID
	if localID == "" {
		var err error
		localID, err = s.WriteProjectID()
		if err != nil {
			return nil, err
		}
	}
	local := map[string]bool{localID: true}
	run, err := s.LoadProvenanceReviewRun(runID)
	if err != nil || run.BundleID == "" {
		// A refusal for an unreadable run writes no positive review decision.
		return local, nil
	}
	bundle, err := s.LoadReviewBundle(run.BundleID)
	if err != nil {
		return local, nil
	}
	_, projects, err := s.reviewBundleProjectSet(bundle.Payload.Scope.Ref)
	if err != nil {
		// The conclusion path will record this as a refusal. If a later guarded
		// read can establish a wider set, the caller's set comparison stops it.
		return local, nil
	}
	return projects, nil
}

func (s Store) withReviewRunWriteGuard(operation, runID string, callback func(Store) error) error {
	lockedProjects, err := s.reviewRunProjectSet(runID)
	if err != nil {
		return err
	}
	return s.withExplicitGovernanceWriteGuard(operation, governanceProjectIDs(lockedProjects), func(guarded Store) error {
		observedProjects, err := guarded.reviewRunProjectSet(runID)
		if err != nil {
			return err
		}
		if err := requireSameGovernanceProjectSet(operation, lockedProjects, observedProjects); err != nil {
			return err
		}
		return callback(guarded)
	})
}

func (s Store) withReviewRunWriteGuardRetry(operation, runID string, callback func(Store) error) error {
	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		err := s.withReviewRunWriteGuard(operation, runID, callback)
		if !errors.Is(err, ErrGovernanceWriteGuardBusy) || !time.Now().Before(deadline) {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
}
