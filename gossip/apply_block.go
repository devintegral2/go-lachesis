package gossip

import (
	"github.com/Fantom-foundation/go-lachesis/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"math"

	"github.com/Fantom-foundation/go-lachesis/evm_core"
	"github.com/Fantom-foundation/go-lachesis/inter"
	"github.com/Fantom-foundation/go-lachesis/inter/pos"
)

// onNewBlock execs ordered txns of new block on state.
func (s *Service) onNewBlock(
	block *inter.Block,
	stateHash common.Hash,
	validators pos.Validators,
) (
	newStateHash common.Hash,
	newValidators pos.Validators,
) {
	evmProcessor := evm_core.NewStateProcessor(params.AllEthashProtocolChanges, s.GetEvmStateReader())

	// Assemble block data
	evmHeader := evm_core.ToEvmHeader(block)
	evmBlock := &evm_core.EvmBlock{
		EvmHeader:    *evmHeader,
		Transactions: make(types.Transactions, 0, len(block.Events)*10),
	}
	txPositions := make(map[common.Hash]TxPosition)
	for _, id := range block.Events {
		e := s.store.GetEvent(id)
		if e == nil {
			s.Log.Crit("Event not found", "event", id.String())
		}

		evmBlock.Transactions = append(evmBlock.Transactions, e.Transactions...)
		for i, tx := range e.Transactions {
			// we don't care if tx was met in different events, any valid position will work
			txPositions[tx.Hash()] = TxPosition{
				Event:       e.Hash(),
				EventOffset: uint32(i),
			}
		}
	}
	s.occurredTxs.CollectConfirmedTxs(evmBlock.Transactions) // TODO collect all the confirmed txs, not only block txs

	// Process txs
	statedb := s.store.StateDB(stateHash)
	receipts, _, gasUsed, totalFee, skipped, err := evmProcessor.Process(evmBlock, statedb, vm.Config{}, false)
	if err != nil {
		s.Log.Crit("Shouldn't happen ever because it's not strict", "err", err)
	}
	block.SkippedTxs = skipped
	block.GasUsed = gasUsed

	// apply block rewards here if needed
	log.Info("New block", "index", block.Index, "hash", block.Hash().String(), "fee", totalFee, "txs", len(evmBlock.Transactions), "skipped_txs", len(skipped))

	// finalize
	newStateHash, err = statedb.Commit(true)
	if err != nil {
		s.Log.Crit("Failed to commit state", "err", err)
	}
	block.Root = newStateHash
	evmBlock.Root = newStateHash
	s.store.SetBlock(block)
	s.store.SetBlockIndex(block.Hash(), block.Index)

	// new validators
	// TODO replace with special transactions for changing validators state
	// TODO the schema below doesn't work in all the cases, and intended only for testing
	{
		newValidators = validators.Copy()
		for addr := range validators.Iterate() {
			stake := pos.BalanceToStake(statedb.GetBalance(addr))
			newValidators.Set(addr, stake)
		}
		for _, tx := range evmBlock.Transactions {
			if tx.To() == nil {
				continue
			}
			stake := pos.BalanceToStake(statedb.GetBalance(*tx.To()))
			newValidators.Set(*tx.To(), stake)
		}
	}

	// Filter skipped transactions before notifying. Receipts are filtered already
	skipCount := 0
	filteredTxs := make(types.Transactions, 0, len(evmBlock.Transactions))
	for i, tx := range evmBlock.Transactions {
		if skipCount < len(block.SkippedTxs) && block.SkippedTxs[skipCount] == uint(i) {
			skipCount++
		} else {
			filteredTxs = append(filteredTxs, tx)
		}
	}
	evmBlock.Transactions = filteredTxs

	// Build index for not skipped txs
	if s.config.TxIndex {
		for i, tx := range evmBlock.Transactions {
			// not skipped txs only
			position := txPositions[tx.Hash()]
			position.Block = block.Index
			position.BlockOffset = uint32(i)
			s.store.SetTxPosition(tx.Hash(), &position)
		}

		if receipts.Len() != 0 {
			s.store.SetReceipts(block.Index, receipts)
		}
	}

	// Calc validators score
	s.store.SetBlockGasUsed(block.Index, block.GasUsed)
	for v := range validators.Iterate() {
		// Check validator events in current block
		eventsInBlock := false
		for _, evHash := range block.Events {
			evh := s.store.GetEventHeader(evHash.Epoch(), evHash)
			if evh.Creator == v {
				eventsInBlock = true
				break
			}
		}

		// If have not events in block - add missed blocks for validator
		if !eventsInBlock {
			s.store.IncBlocksMissed(v)
			continue
		}

		missed := s.store.GetBlocksMissed(v)
		s.store.AddDirtyValidatorsScore(v, block.GasUsed)
		for i := 1; i < int(math.Min(2, float64(missed))); i++ {
			usedGas := s.store.GetBlockGasUsed(block.Index - idx.Block(i))
			s.store.AddDirtyValidatorsScore(v, usedGas)
		}
		s.store.ResetBlocksMissed(v)
	}

	// Notify about new block
	s.feed.newBlock.Send(evm_core.ChainHeadNotify{evmBlock})

	// Flush trie on the main DB
	err = statedb.Database().TrieDB().Cap(0)
	if err != nil {
		log.Error("Failed to flush trie DB into main DB", "err", err)
	}

	return newStateHash, newValidators
}
