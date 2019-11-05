package gossip

import (
	"github.com/Fantom-foundation/go-lachesis/common/bigendian"
	"github.com/ethereum/go-ethereum/common"
)

func (s *Store) saveDelagationData(key []byte, val uint64) {
	valBytes := bigendian.Int64ToBytes(val)

	err := s.table.Delegates.Put(key, valBytes)
	if err != nil {
		s.Log.Crit("Failed to set key-value", "err", err)
	}
}

// SetDelegation save delegation value from delegator to validator
func (s *Store) SetDelegation(validator, delegator common.Address, val uint64) {
	// Calc difference
	valDiff := val - s.GetDelegationValue(validator, delegator)

	// Save main data
	key := []byte(validator.String()+":"+delegator.String())
	s.saveDelagationData(key, val)

	// Save data for delegator
	oldVal := s.GetDelegatorValue(delegator)
	val = oldVal + valDiff
	key = []byte(":"+delegator.String())
	s.saveDelagationData(key, val)

	// Save data for validator
	oldVal = s.GetValidatorDelegationValue(validator)
	val = oldVal + valDiff
	key = []byte(validator.String()+":")
	s.saveDelagationData(key, val)
}

func (s *Store) GetDelegationValue(validator, delegator common.Address) uint64 {
	key := []byte(validator.String()+":"+delegator.String())

	valBytes, err := s.table.Delegates.Get(key)
	if err != nil {
		s.Log.Crit("Failed to get key-value", "err", err)
	}

	val := bigendian.BytesToInt64(valBytes)

	return val
}

func (s *Store) GetDelegatorValue(delegator common.Address) uint64 {
	key := []byte(":"+delegator.String())

	valBytes, err := s.table.Delegates.Get(key)
	if err != nil {
		s.Log.Crit("Failed to get key-value", "err", err)
	}

	val := bigendian.BytesToInt64(valBytes)

	return val
}

func (s *Store) GetValidatorDelegationValue(validator common.Address) uint64 {
	key := []byte(validator.String()+":")

	valBytes, err := s.table.Delegates.Get(key)
	if err != nil {
		s.Log.Crit("Failed to get key-value", "err", err)
	}

	val := bigendian.BytesToInt64(valBytes)

	return val
}
