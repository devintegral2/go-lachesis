package tendermint

import (
	abcitypes "github.com/tendermint/tendermint/abci/types"
)

type Stub struct{}

var _ abcitypes.Application = (*Stub)(nil)

// NewStubApplication constructor.
func NewStubApplication() *Stub {
	return &Stub{}
}

// Info returns information about the application state.
func (Stub) Info(req abcitypes.RequestInfo) abcitypes.ResponseInfo {
	return abcitypes.ResponseInfo{}
}

// SetOption sets non-consensus critical application specific options.
func (Stub) SetOption(req abcitypes.RequestSetOption) abcitypes.ResponseSetOption {
	return abcitypes.ResponseSetOption{}
}

// InitChain will be called once upon genesis.
func (Stub) InitChain(req abcitypes.RequestInitChain) abcitypes.ResponseInitChain {
	return abcitypes.ResponseInitChain{}
}

// CheckTx validates a mempool transaction, prior to broadcasting or proposing.
func (Stub) CheckTx(req abcitypes.RequestCheckTx) abcitypes.ResponseCheckTx {
	return abcitypes.ResponseCheckTx{Code: 0}
}

// Query for data from the application at current or past height.
// Optionally returns Merkle proof.
func (Stub) Query(req abcitypes.RequestQuery) abcitypes.ResponseQuery {
	return abcitypes.ResponseQuery{Code: 0}
}

// BeginBlock signals the beginning of a new block. Called prior to any DeliverTxs.
func (Stub) BeginBlock(req abcitypes.RequestBeginBlock) abcitypes.ResponseBeginBlock {
	// The header is expected to at least contain the Height.
	// The Validators and ByzantineValidators can be used to determine rewards and punishments for the validators.
	return abcitypes.ResponseBeginBlock{}
}

// DeliverTx delivers a transaction to be executed in full by the application. If the transaction is valid, returns CodeType.OK.
func (Stub) DeliverTx(req abcitypes.RequestDeliverTx) abcitypes.ResponseDeliverTx {
	// Keys and values in Tags must be UTF-8 encoded strings (e.g. “account.owner”: “Bob”, “balance”: “100.0”, “time”: “2018-01-02T12:30:00Z”)
	return abcitypes.ResponseDeliverTx{Code: 0}
}

// EndBlock signals the end of a block.
// Called prior to each Commit, after all transactions.
func (Stub) EndBlock(req abcitypes.RequestEndBlock) abcitypes.ResponseEndBlock {
	// Validator set and consensus params are updated with the result.
	// Validator pubkeys are expected to be go-wire encoded.
	return abcitypes.ResponseEndBlock{}
}

// Commit the application state.
// Returns a Merkle root hash of the application state.
func (Stub) Commit() abcitypes.ResponseCommit {
	return abcitypes.ResponseCommit{}
}
