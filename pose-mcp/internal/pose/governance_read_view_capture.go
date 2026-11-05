package pose

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	// ErrGovernanceReadViewChanged means the captured input closure changed
	// during both the original read and its one permitted retry.
	ErrGovernanceReadViewChanged = errors.New("pose: governance read view changed during capture")
	// ErrGovernanceReadViewUnknown means the operation could not prove the
	// completeness of its captured input closure.
	ErrGovernanceReadViewUnknown = errors.New("pose: governance read view input closure is unknown")
)

// WithGovernanceReadView runs a read-only callback against one immutable view
// and audits that view before returning. A changed view is retried once; if
// the retry changes or any required observation is unknown, the callback must
// not be used to publish a derived result. The callback must keep its outputs
// in memory; publishers write through the caller's original guarded Store only
// after this method succeeds.
func (s Store) WithGovernanceReadView(producer string, run func(Store) error) error {
	if run == nil {
		return errors.New("pose: governance read view callback is required")
	}
	if producer == "" {
		return errors.New("pose: governance read view producer is required")
	}
	return withGovernanceReadViewCaptureAttempt(s.withoutGovernanceReadView(), producer, 0, false, func(store Store) (bool, error) {
		return true, run(store)
	})
}

// WithGovernanceReadViewForIndexBuild allows the callback to discover whether
// the index inputs depend on a Git source revision. A missing primary HEAD is
// captured and audited as data; the callback returns true when that revision
// is semantically required by the generation it produced.
func (s Store) WithGovernanceReadViewForIndexBuild(producer string, run func(Store) (requiresPrimaryRevision bool, err error)) error {
	if run == nil {
		return errors.New("pose: governance read view callback is required")
	}
	if producer == "" {
		return errors.New("pose: governance read view producer is required")
	}
	return withGovernanceReadViewCaptureAttempt(s.withoutGovernanceReadView(), producer, 0, true, run)
}

func withGovernanceReadViewCaptureAttempt(s Store, producer string, attempt int, allowMissingPrimaryRevision bool, run func(Store) (bool, error)) error {
	base, view := s.beginGovernanceReadView(allowMissingPrimaryRevision)
	store := base.withGovernanceReadView(view, producer).PinIndexes()
	if !allowMissingPrimaryRevision {
		_, _ = captureGovernanceSourceRevision(store.fsys(), store.Root, producer)
	}
	requiresPrimaryRevision, err := run(store)
	if err != nil {
		_ = view.audit()
		return err
	}
	if requiresPrimaryRevision {
		view.operation.requirePrimaryRevision(producer)
	}
	audit := view.audit()
	changed := governanceReviewAuditChanged(audit)
	if changed && attempt == 0 {
		return withGovernanceReadViewCaptureAttempt(base.withoutGovernanceReadView(), producer, 1, allowMissingPrimaryRevision, run)
	}
	if changed {
		return ErrGovernanceReadViewChanged
	}
	if len(audit.UnknownProducers) > 0 {
		producers := make([]string, 0, len(audit.UnknownProducers))
		for producer := range audit.UnknownProducers {
			producers = append(producers, producer)
		}
		sort.Strings(producers)
		clip := func(value string, limit int) string {
			if len(value) <= limit {
				return value
			}
			return value[:limit-3] + "..."
		}
		var summary strings.Builder
		for i, producer := range producers {
			if i >= 8 {
				fmt.Fprintf(&summary, "; ... %d more producer(s)", len(producers)-i)
				break
			}
			if i > 0 {
				summary.WriteString("; ")
			}
			summary.WriteString(clip(producer, 128))
			summary.WriteString(": ")
			reasons := append([]string(nil), audit.UnknownProducers[producer]...)
			sort.Strings(reasons)
			for j, reason := range reasons {
				if j >= 4 {
					fmt.Fprintf(&summary, "; ... %d more reason(s)", len(reasons)-j)
					break
				}
				if j > 0 {
					summary.WriteString(" | ")
				}
				summary.WriteString(clip(reason, 256))
			}
		}
		detail := summary.String()
		if len(detail) > 2048 {
			detail = detail[:2045] + "..."
		}
		return fmt.Errorf("%w: %d producer(s) lack a complete observation: %s", ErrGovernanceReadViewUnknown, len(audit.UnknownProducers), detail)
	}
	return nil
}
