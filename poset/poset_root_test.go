package poset

import (
	"github.com/Fantom-foundation/go-lachesis/hash"
	"github.com/ethereum/go-ethereum/common"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Fantom-foundation/go-lachesis/inter"
	"github.com/Fantom-foundation/go-lachesis/inter/idx"
)

var (
	testScheme = `
 A1.01    
 ║         ║        
 ╠════════ B1.01    
 ║         ║         ║        
 ╠════════─╫─═══════ C1.01    
 ║         ║         ║         ║        
 ╠════════─╫─═══════─╫─═══════ D1.01    
 ║         ║         ║         ║        
 a1.02════─╫─═══════─╫─════════╣        
 ║         ║         ║         ║        
 ║         b1.02════─╫─════════╣        
 ║         ║         ║         ║        
 ║         ║         c1.02═════╣        
 ║         ║         ║         ║        
 a1.03════─╫─════════╣         ║        
 ║         ║         ║         ║        
 ╠════════ B2.03     ║         ║        
 ║         ║║        ║         ║        
 ║         ║╚═══════─╫─═══════ d1.02    
 ║         ║         ║         ║        
 ║         ║         C2.03═════╣        
 ║         ║         ║         ║        
 A2.04════─╫─════════╣         ║        
 ║         ║         ║         ║        
 ║         b2.04═════╣         ║        
 ║         ║║        ║         ║        
 ║         ║╚═══════─╫─═══════ D2.03    
 ║         ║         ║         ║        
 ║         ║         c2.04═════╣        
 ║         ║         ║         ║        
 ║         ║         ╠════════ d2.04    
 ║         ║         ║         ║        
 A3.05════─╫─═══════─╫─════════╣        
 ║         ║         ║         ║        
 ╠════════ B3.05     ║         ║        
 ║         ║         ║         ║        
 ║         ╠════════ C3.05     ║        
 ║         ║         ║         ║        
 ║         ╠════════─╫─═══════ D3.05    
 ║         ║         ║         ║        
 a3.06════─╫─═══════─╫─════════╣        
 ║         ║         ║         ║        
 ║         b3.06════─╫─════════╣        
 ║         ║         ║         ║        
 ║         ║         c3.06═════╣        
 ║         ║         ║         ║        
 ║         B4.07═════╣         ║        
 ║         ║         ║         ║        
 ║         ║         ╠════════ d3.06    
 ║         ║         ║         ║        
 A4.07════─╫─═══════─╫─════════╣        
 ║         ║         ║         ║        
 a4.08═════╣         ║         ║        
 ║║        ║         ║         ║        
 ║╚═══════─╫─═══════ C4.07     ║        
 ║         ║         ║         ║        
 ║         b4.08═════╣         ║        
 ║         ║         ║         ║        
 a4.09═════╣         ║         ║        
 ║3        ║         ║         ║        
 ║╚═══════─╫─═══════─╫─═══════ D4.07    
 ║         ║         ║         ║        
 ║         ║         c4.08═════╣        
 ║         ║         ║         ║        
 ║         b4.09═════╣         ║        
 ║         ║         ║         ║        
 ║         ╠════════ c4.09     ║        
 ║         ║         ║         ║        
 A5.10════─╫─════════╣         ║        
 ║         ║         ║         ║        
 ╠════════ B5.10     ║         ║        
 ║         ║3        ║         ║        
 ║         ║╚═══════─╫─═══════ d4.08    
 ║║        ║         ║         ║        
 ║╚═══════─╫─═══════─╫─═══════ D5.09    
 ║         ║         ║         ║        
 ║         ║         C5.10═════╣        
 ║         ║         ║         ║        
 ╠════════─╫─═══════─╫─═══════ d5.10    
 ║         ║         ║         ║        
 a5.11════─╫─═══════─╫─════════╣        
 ║         ║         ║         ║        
 ╠════════ b5.11     ║         ║        
 ║         ║         ║         ║        
 ║         ╠════════ c5.11     ║        
 ║         ║         ║         ║        
 A6.12════─╫─════════╣         ║        
 ║         ║         ║         ║        
 ║         ╠════════─╫─═══════ d5.11    
 ║         ║         ║         ║        
 ║         b5.12════─╫─════════╣        
 ║         ║         ║         ║        
 ║         ╠════════ C6.12     ║        
 ║         ║         ║         ║        
 ╠════════─╫─═══════─╫─═══════ D6.12    
 ║         ║         ║         ║        
 a6.13════─╫─═══════─╫─════════╣        
 ║         ║         ║         ║        
 ║         B6.13════─╫─════════╣        
 ║         ║         ║         ║        
 a6.14═════╣         ║         ║        
 ║║        ║         ║         ║        
 ║╚═══════─╫─═══════ c6.13     ║        
 ║         ║         ║         ║        
 ╠════════─╫─═══════ C7.14     ║        
 ║║        ║         ║         ║        
 ║╚═══════─╫─═══════─╫─═══════ d6.13    
 ║         ║         ║         ║        
 ║         b6.14════─╫─════════╣        
 ║         ║         ║         ║        
 a6.15═════╣         ║         ║        
 ║         ║         ║         ║        
 ║         B7.15═════╣         ║        
 ║         ║║        ║         ║        
 ║         ║╚═══════─╫─═══════ d6.14    
 ║         ║         ║         ║        
 ║         ║         c7.15═════╣        
 ║         ║         ║         ║        
 ╠════════─╫─═══════─╫─═══════ D7.15    
 ║         ║         ║         ║        
 A7.16════─╫─═══════─╫─════════╣        
 ║         ║         ║         ║        
 ║         b7.16════─╫─════════╣        
 ║         ║         ║         ║        
 ║         ║         c7.16═════╣        
 ║         ║         ║         ║        
 a7.17════─╫─════════╣         ║        
 ║         ║         ║         ║        
 ║         ║         ╠════════ d7.16    
 ║         ║         ║         ║        
 ║         b7.17════─╫─════════╣        
 ║         ║         ║         ║        
 ║         ║         c7.17═════╣        
 ║         ║         ║         ║        
 a7.18════─╫─════════╣         ║        
 ║         ║         ║         ║        
 ╠════════─╫─═══════ c7.18     ║        
 ║║        ║         ║         ║        
 ║╚═══════─╫─═══════─╫─═══════ d7.17    
 ║         ║         ║         ║        
 ║         B8.18════─╫─════════╣        
 ║         ║         ║         ║        
 ║         b8.19═════╣         ║        
 ║         ║║        ║         ║        
 ║         ║╚═══════─╫─═══════ D8.18    
 ║         ║         ║         ║        
 A8.19════─╫─═══════─╫─════════╣        
 ║         ║         ║         ║        
 ╠════════─╫─═══════ C8.19     ║        
 ║         ║         ║         ║        
 ╠════════─╫─═══════─╫─═══════ d8.19    
 ║         ║         ║         ║        
 a8.20════─╫─═══════─╫─════════╣        
 ║         ║         ║         ║        
 ║         B9.20════─╫─════════╣        
 ║         ║         ║         ║        
 ║         ║         C9.20═════╣        
 ║         ║         ║         ║        
 ║         ║         ╠════════ D9.20   
`
)

func TestPosetClassicRoots(t *testing.T) {
	testSpecialNamedRoots(t, `
A1.01  B1.01  C1.01  D1.01  // 1
║      ║      ║      ║
║      ╠──────╫───── d1.02
║      ║      ║      ║
║      b1.02 ─╫──────╣
║      ║      ║      ║
║      ╠──────╫───── d1.03
a1.02 ─╣      ║      ║
║      ║      ║      ║
║      b1.03 ─╣      ║
║      ║      ║      ║
║      ╠──────╫───── d1.04
║      ║      ║      ║
║      ╠───── c1.02  ║
║      ║      ║      ║
║      b1.04 ─╫──────╣
║      ║      ║      ║     // 2
╠──────╫──────╫───── D2.05
║      ║      ║      ║
A2.03 ─╫──────╫──────╣
║      ║      ║      ║
a2.04 ─╫──────╣      ║
║      ║      ║      ║
║      B2.05 ─╫──────╣
║      ║      ║      ║
║      ╠──────╫───── d2.06
a2.05 ─╣      ║      ║
║      ║      ║      ║
╠──────╫───── C2.03  ║
║      ║      ║      ║
╠──────╫──────╫───── d2.07
║      ║      ║      ║
╠───── b2.06  ║      ║
║      ║      ║      ║     // 3
║      B3.07 ─╫──────╣
║      ║      ║      ║
A3.06 ─╣      ║      ║
║      ╠──────╫───── D3.08
║      ║      ║      ║
║      ║      ╠───── d309
╠───── b3.08  ║      ║
║      ║      ║      ║
╠───── b3.09  ║      ║
║      ║      C3.04 ─╣
a3.07 ─╣      ║      ║
║      ║      ║      ║
║      b3.10 ─╫──────╣
║      ║      ║      ║
a3.08 ─╣      ║      ║
║      ╠──────╫───── d3.10
║      ║      ║      ║
╠───── b3.11  ║      ║     // 4
║      ║      ╠───── D4.11
║      ║      ║      ║
║      B4.12 ─╫──────╣
║      ║      ║      ║
`)
}

func TestPosetRandomRoots(t *testing.T) {
	// generated by codegen4PosetRandomRoot()
	testSpecialNamedRoots(t, testScheme)
}

/*
 * Utils:
 */

// testSpecialNamedRoots is a general test of root selection.
// Event name means:
// - 1st letter uppercase - event should be root;
// - 2nd number - frame where event should be in;
// - "." - separator;
// - tail - makes name unique;
func testSpecialNamedRoots(t *testing.T, scheme string) {
	//logger.SetTestMode(t)
	assertar := assert.New(t)

	// decode is a event name parser
	decode := func(name string) (frameN idx.Frame, isRoot bool) {
		n, err := strconv.ParseUint(strings.Split(name, ".")[0][1:2], 10, 64)
		if err != nil {
			panic(err.Error() + ". Name event " + name + " properly: <UpperCaseForRoot><FrameN><Index>")
		}
		frameN = idx.Frame(n)

		isRoot = name == strings.ToUpper(name)
		return
	}

	// get nodes only
	nodes, _, _ := inter.ASCIIschemeToDAG(scheme)
	// init poset
	p, _, input := FakePoset("", nodes)

	// process events
	_, _, names := inter.ASCIIschemeForEach(scheme, inter.ForEachEvent{
		Process: func(e *inter.Event, name string) {
			input.SetEvent(e)
			assertar.NoError(
				p.ProcessEvent(e))
			assertar.NoError(
				flushDb(p, e.Hash()))
		},
		Build: func(e *inter.Event, name string) *inter.Event {
			e.Epoch = p.GetEpoch()
			e = p.Prepare(e)

			return e
		},
	})

	// check each
	for name, event := range names {
		mustBeFrame, mustBeRoot := decode(name)
		// check root
		frame := p.GetEventHeader(p.EpochN, event.Hash()).Frame
		isRoot := p.store.IsRoot(frame, event.Creator, event.Hash())
		if !assertar.Equal(mustBeRoot, isRoot, name+" is root") {
			break
		}
		if !assertar.Equal(mustBeRoot, event.IsRoot, name+" is root") {
			break
		}
		// check frame
		if !assertar.Equal(idx.Frame(mustBeFrame), frame, "frame of "+name) {
			break
		}
	}
}

/*
// codegen4PosetRandomRoot is for test data generation.
func codegen4PosetRandomRoot() {
	nodes, events := inter.GenEventsByNode(4, 20, 2, nil)

	p, _, input := FakePoset(nodes)
	// process events
	dag := inter.Events{}
	for _, ee := range events {
		dag = append(dag, ee...)
		for _, e := range ee {
			input.SetEvent(e)
			p.PushEventSync(e.Hash())
		}
	}

	// set event names
	for _, e := range dag {
		frame := p.FrameOfEvent(e.Hash())
		_, isRoot := frame.Roots[e.Creator][e.Hash()]
		oldName := hash.GetEventName(e.Hash())
		newName := fmt.Sprintf("%s%d.%02d", oldName[0:1], frame.Index, e.Seq)
		if isRoot {
			newName = strings.ToUpper(newName[0:1]) + newName[1:]
		}
		hash.SetEventName(e.Hash(), newName)
	}

	fmt.Println(inter.DAGtoASCIIscheme(dag))
}
*/

// TODO: Create benchmark test for parallel & nonparallel variants
func forklessCausedByQuorumOn(parallel bool, p *Poset, e *inter.Event, f idx.Frame) bool {
	observedCounter := p.Validators.NewCounter()
	// check "observing" prev roots only if called by creator, or if creator has marked that event as root
	// lock := &sync.Mutex{}
	p.store.ForEachRoot(parallel, f, func(f idx.Frame, from common.Address, root hash.Event) bool {
		// lock.Lock()
		// defer lock.Unlock()

		if p.vecClock.ForklessCause(e.Hash(), root) {
			observedCounter.Count(from)
		}
		return !observedCounter.HasQuorum()
	})
	return observedCounter.HasQuorum()
}

func BenchmarkStore_ForEachRoot(b *testing.B) {
	nodes, events, _ := inter.ASCIIschemeToDAG(testScheme)
	p, _, _ := FakePoset("", nodes)

	b.Run("parallel", func(b *testing.B){
		e := events[nodes[0]][len(events[nodes[0]])-1]
		for i := 0; i < b.N; i++ {
			forklessCausedByQuorumOn(true, p.Poset, e, e.Frame)
		}
	})
	b.Run("sequenced", func(b *testing.B){
		e := events[nodes[0]][len(events[nodes[0]])-1]
		for i := 0; i < b.N; i++ {
			forklessCausedByQuorumOn(false, p.Poset, e, e.Frame)
		}
	})
}
