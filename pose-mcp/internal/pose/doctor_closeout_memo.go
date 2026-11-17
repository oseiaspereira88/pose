package pose

import (
	"sync"
	"time"
)

const (
	doctorMemoMaxEntries      = 2048
	doctorMemoMaxVisited      = 256
	doctorMemoMaxTotalVisited = 32768
)

// This memo belongs only to CloseoutReviewsOwed's callback, not to the Store
// or even to the surrounding review operation. It retains only approvals:
// terminal verification also turns some errors into false, so false is never
// cached. An incomplete trace, a cycle or a project crossing disqualifies all
// enclosing computations. Limits fall back to the original verifier.
type doctorCloseoutMemo struct {
	mu      sync.Mutex
	view    *reviewOperationView
	at      time.Time
	closed  bool
	entries map[string]doctorCloseoutEntry
	visited int
	// Test observation of real verifier invocations; never substitutes a result.
	onCompute func(string)
}

type doctorCloseoutEntry struct{ visited []string }

type doctorCloseoutTrace struct {
	mu      sync.Mutex
	visited map[string]bool
	cut     bool
}

// AggregateDependencyClosed traces aggregate crossings. Remote spec proofs
// also use source stores, with their own clocks; exclude them before retaining
// an approval. The manifest keeps the named identity, so resolve its canonical
// destination too (a local name can redirect to another project). This reads
// through the same bound resolver snapshots and capture as the verifier.
func (s Store) traceDoctorCloseoutSources(state CloseoutState) {
	if s.closeoutTrace == nil || !state.Terminal || state.FederatedAcceptance == nil {
		return
	}
	for _, dep := range state.FederatedAcceptance.Manifest.Dependencies {
		resolved := s.FederatedResolver.Resolve(s.FederatedProjectID, dep.Ref)
		identity := resolved.Identity
		if resolved.CanonicalIdentity != nil {
			identity = *resolved.CanonicalIdentity
		}
		if !resolved.Resolved || identity.Project != s.FederatedProjectID {
			s.closeoutTrace.visit("", true)
			return
		}
	}
}

func (t *doctorCloseoutTrace) visit(scope string, cut bool) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cut = t.cut || cut
	if scope == "" || t.visited[scope] {
		return
	}
	if len(t.visited) >= doctorMemoMaxVisited {
		t.cut = true
		return
	}
	t.visited[scope] = true
}

func (t *doctorCloseoutTrace) snapshot() ([]string, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	visited := make([]string, 0, len(t.visited))
	for scope := range t.visited {
		visited = append(visited, scope)
	}
	return visited, t.cut
}

func (t *doctorCloseoutTrace) merge(visited []string, cut bool) {
	t.visit("", cut)
	for _, scope := range visited {
		t.visit(scope, false)
	}
}

func (s Store) withDoctorCloseoutMemo(run func(Store)) {
	// Local review expiry/trust checks in this diagnostic observe the instant
	// the query starts (or its explicitly supplied review clock). They still
	// use current authority and the normal final input audit. The next query
	// gets a fresh clock and capture; this is no extension of a review budget.
	// Remote proofs keep their original clock handling and are not memoized.
	s.doctorCloseoutMemo, s.closeoutTrace = nil, nil
	if s.reviewView == nil || !s.reviewView.boundTo(s) ||
		s.FederatedResolver == nil || s.FederatedResolver.Roots == nil ||
		s.FederatedResolver.Authorize != nil {
		// A dynamic authorization callback must be asked on every traversal.
		run(s)
		return
	}
	// Fallback retains its existing clock semantics, including a live clock.
	s = s.atReviewTime(s.reviewClock())
	m := &doctorCloseoutMemo{view: s.reviewView, at: s.reviewClock(), entries: map[string]doctorCloseoutEntry{}}
	s.doctorCloseoutMemo = m
	defer func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.closed, m.entries, m.visited = true, nil, 0
	}()
	run(s)
}

func (s Store) doctorMemoClosedByReceipt(ref string) (bool, error) {
	m := s.doctorCloseoutMemo
	if m == nil || s.reviewView != m.view || !m.view.boundTo(s) ||
		s.reviewTime == nil || !s.reviewTime.Equal(m.at) || s.indexMaterial || s.planWithoutCloseoutGate ||
		s.FederatedResolver == nil || s.FederatedResolver.Authorize != nil {
		// A child under an incompatible context cannot leave a reusable parent.
		s.closeoutTrace.visit("", true)
		return s.computeClosedByReceipt(ref)
	}
	readView, _ := governanceReadViewFromFiles(s.fsys())
	if readView == nil {
		s.closeoutTrace.visit("", true)
		return s.computeClosedByReceipt(ref)
	}
	readView.mu.Lock()
	finalized := readView.closed
	readView.mu.Unlock()
	if finalized {
		s.closeoutTrace.visit("", true)
		return s.computeClosedByReceipt(ref)
	}
	key := m.view.root + "\x00" + m.view.project + "\x00" + ref
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return s.computeClosedByReceipt(ref)
	}
	entry, hit := m.entries[key]
	if hit {
		for _, ancestor := range s.closeoutChain {
			for _, visited := range entry.visited {
				if ancestor == visited {
					hit = false
					break
				}
			}
			if !hit {
				break
			}
		}
	}
	m.mu.Unlock()
	if hit {
		s.closeoutTrace.merge(entry.visited, false)
		return true, nil
	}
	parent := s.closeoutTrace
	trace := &doctorCloseoutTrace{visited: map[string]bool{}}
	trace.visit(ref, false)
	s.closeoutTrace = trace
	// Never hold a memo/trace lock (or Once) over recursive verification.
	if m.onCompute != nil {
		m.onCompute(ref)
	}
	closed, err := s.computeClosedByReceipt(ref)
	visited, cut := trace.snapshot()
	parent.merge(visited, cut || err != nil)
	if closed && err == nil && !cut {
		m.mu.Lock()
		if _, exists := m.entries[key]; !m.closed && !exists &&
			len(m.entries) < doctorMemoMaxEntries && m.visited+len(visited) <= doctorMemoMaxTotalVisited {
			m.entries[key] = doctorCloseoutEntry{visited: visited}
			m.visited += len(visited)
		}
		m.mu.Unlock()
	}
	return closed, err
}
