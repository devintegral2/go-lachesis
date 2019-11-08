package election

import (
	"fmt"
	"sync"

	"github.com/Fantom-foundation/go-lachesis/hash"
)

var (
	VotesLock = &sync.Mutex{}
	DecidedRootsLock = &sync.Mutex{}
)

// ProcessRoot calculates Atropos votes only for the new root.
// If this root observes that the current election is decided, then @return decided Atropos
func (el *Election) ProcessRoot(newRoot RootAndSlot) (*Res, error) {
	DecidedRootsLock.Lock()
	if len(el.decidedRoots) == el.validators.Len() {
		DecidedRootsLock.Unlock()
		// current election is already decided
		return el.chooseAtropos()
	}
	DecidedRootsLock.Unlock()

	if newRoot.Slot.Frame <= el.frameToDecide {
		// too old root, out of interest for current election
		return nil, nil
	}
	round := newRoot.Slot.Frame - el.frameToDecide

	notDecidedRoots := el.notDecidedRoots()
	for _, validatorSubject := range notDecidedRoots {
		vote := voteValue{}

		if round == 1 {
			// in initial round, vote "yes" if observe the subject
			observedRoot := el.observe(newRoot.Root, validatorSubject, el.frameToDecide)
			vote.yes = observedRoot != nil
			vote.decided = false
			if observedRoot != nil {
				vote.observedRoot = *observedRoot
			}
		} else if round > 1 {
			observedRoots := el.observedRoots(newRoot.Root, newRoot.Slot.Frame-1)

			var (
				yesVotes = el.validators.NewCounter()
				noVotes  = el.validators.NewCounter()
				allVotes = el.validators.NewCounter()
			)

			// calc number of "yes" and "no", weighted by validator's stake
			var subjectHash *hash.Event
			for _, observedRoot := range observedRoots {
				vid := voteID{
					forValidator: validatorSubject,
					fromRoot:     observedRoot.Root,
				}

				VotesLock.Lock()
				if vote, ok := el.votes[vid]; ok {
					VotesLock.Unlock()
					if vote.yes && subjectHash != nil && *subjectHash != vote.observedRoot {
						return nil, fmt.Errorf("forkless caused by 2 fork roots => more than 1/3W are Byzantine (%s != %s, election frame=%d, validator=%s)",
							subjectHash.String(), vote.observedRoot.String(), el.frameToDecide, validatorSubject.String())
					}

					if vote.yes {
						subjectHash = &vote.observedRoot
						yesVotes.Count(observedRoot.Slot.Addr)
					} else {
						noVotes.Count(observedRoot.Slot.Addr)
					}
					if !allVotes.Count(observedRoot.Slot.Addr) {
						// it shouldn't be possible to get here, because we've taken 1 root from every node above
						return nil, fmt.Errorf("forkless caused by 2 fork roots => more than 1/3W are Byzantine (%s, election frame=%d, validator=%s)",
							subjectHash.String(), el.frameToDecide, validatorSubject.String())
					}
				} else {
					VotesLock.Unlock()
					el.Log.Crit("Every root must vote for every not decided subject. Possibly roots are processed out of order",
						"root", newRoot.Root.String())
				}
			}
			// sanity checks
			if !allVotes.HasQuorum() {
				el.Log.Crit("Root must be forkless caused by at least 2/3W of prev roots. Possibly roots are processed out of order",
					"root", newRoot.Root.String(),
					"votes", allVotes.Sum())
			}

			// vote as majority of votes
			vote.yes = yesVotes.Sum() >= noVotes.Sum()
			if vote.yes && subjectHash != nil {
				vote.observedRoot = *subjectHash
			}

			// If supermajority is observed, then the final decision may be made.
			// It's guaranteed to be final and consistent unless more than 1/3W are Byzantine.
			vote.decided = yesVotes.HasQuorum() || noVotes.HasQuorum()
			if vote.decided {
				DecidedRootsLock.Lock()
				el.decidedRoots[validatorSubject] = vote
				DecidedRootsLock.Unlock()
			}
		} else {
			continue // we shouldn't be here, we checked it above the loop
		}
		// save vote for next rounds
		vid := voteID{
			fromRoot:     newRoot.Root,
			forValidator: validatorSubject,
		}
		VotesLock.Lock()
		el.votes[vid] = vote
		VotesLock.Unlock()
	}

	DecidedRootsLock.Lock()
	frameDecided := len(el.decidedRoots) == el.validators.Len()
	DecidedRootsLock.Unlock()
	if frameDecided {
		return el.chooseAtropos()
	}

	return nil, nil
}
