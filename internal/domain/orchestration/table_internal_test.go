package orchestration

import "testing"

// This is the only internal test in the package. Everything else exercises the
// public contract from orchestration_test, which is where behaviour belongs.
//
// The table itself, though, is unexported, and there is one class of mistake
// the public tests structurally cannot see: an entry that is dead. Next checks
// IsTerminal before it consults the table, and AllowedReasons short-circuits on
// terminal too, so an edge keyed on Completed, Cancelled or Failed is never
// read -- adding one changes no observable behaviour and fails no public test.
// It is harmless to run and actively misleading to read, because the table is
// documented as the single authority on the lifecycle and would then be
// describing a move that cannot happen.
//
// The same goes for a key or target that is not a declared enum value. Only a
// test inside the package can look, so this one does.

func TestTransitionTableHasNoDeadEntries(t *testing.T) {
	for from, edges := range transitions {
		if !from.IsValid() {
			t.Errorf("the table has a source state that is not a declared state: %d", uint8(from))
			continue
		}
		if from.IsTerminal() {
			t.Errorf("the table has an edge out of terminal state %v; Next never reads it, "+
				"so it is dead weight that claims a move the lifecycle does not model", from)
		}
		if len(edges) == 0 {
			t.Errorf("the table has an entry for %v with no edges; drop the entry instead", from)
		}

		for reason, to := range edges {
			if !reason.IsValid() {
				t.Errorf("%v accepts a reason that is not a declared event: %d", from, uint8(reason))
			}
			if !to.IsValid() {
				t.Errorf("%v on %v leads to a state that is not a declared state: %d", from, reason, uint8(to))
			}
			if to == from {
				t.Errorf("%v on %v leads back to itself; the lifecycle models no self-loop", from, reason)
			}
		}
	}
}

func TestTransitionTableCoversEveryLiveState(t *testing.T) {
	// A live state with no entry at all is a run that can never leave it. The
	// public reachability test proves this of the diagram transcription; this
	// proves it of the table that actually decides.
	for _, state := range States() {
		if state.IsTerminal() {
			continue
		}
		if len(transitions[state]) == 0 {
			t.Errorf("%v is not terminal but the table gives it no outgoing edge", state)
		}
	}
}
