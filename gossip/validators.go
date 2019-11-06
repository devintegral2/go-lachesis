package gossip

import (
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/Fantom-foundation/go-lachesis/common/bigendian"
)

const (
	PoiPeriodDuration = 2 * 24 * time.Hour
)

// PoiPeriod calculate POI period from int64 unix time
func PoiPeriod(t int64) uint64 {
	return uint64(t / int64(PoiPeriodDuration))
}

// CalcValidatorsPOI calculate and save POI for validator
func (s *Store) CalcValidatorsPOI(validator, delegator common.Address, poiPeriod uint64) {
	vGasUsed := s.GetAddressGasUsed(validator)
	dGasUsed := s.GetAddressGasUsed(delegator)

	vGasUsed += dGasUsed
	s.SetAddressGasUsed(validator, vGasUsed)

	poi := uint64((vGasUsed * 1000000) / s.GetPOIGasUsed(poiPeriod))
	s.SetValidatorPOI(validator, poi)
}

// GetAddressGasUsed get gas used by address
func (s *Store) GetAddressGasUsed(addr common.Address) uint64 {
	gasBytes, err := s.table.AddressGasUsed.Get(addr.Bytes())
	if err != nil {
		s.Log.Crit("Failed to get key", "err", err)
	}

	gas := bigendian.BytesToInt64(gasBytes)

	return gas
}

// SetAddressGasUsed save gas used by address
func (s *Store) SetAddressGasUsed(addr common.Address, gas uint64) {
	gasBytes := bigendian.Int64ToBytes(gas)

	err := s.table.AddressGasUsed.Put(addr.Bytes(), gasBytes)
	if err != nil {
		s.Log.Crit("Failed to set key", "err", err)
	}
}

// GetAddressLastTrxTime get last time for last transaction from this address
func (s *Store) GetAddressLastTrxTime(addr common.Address) uint64 {
	gasBytes, err := s.table.AddressLastTrxTime.Get(addr.Bytes())
	if err != nil {
		s.Log.Crit("Failed to get key", "err", err)
	}

	gas := bigendian.BytesToInt64(gasBytes)

	return gas
}

// SetAddressLastTrxTime save last time for trasnaction from this address
func (s *Store) SetAddressLastTrxTime(addr common.Address, gas uint64) {
	gasBytes := bigendian.Int64ToBytes(gas)

	err := s.table.AddressLastTrxTime.Put(addr.Bytes(), gasBytes)
	if err != nil {
		s.Log.Crit("Failed to set key", "err", err)
	}
}

// SetPOIGasUsed save gas used for POI period
func (s *Store) SetPOIGasUsed(poiPeriod uint64, gas uint64) {
	key := bigendian.Int64ToBytes(poiPeriod)
	gasBytes := bigendian.Int64ToBytes(gas)

	err := s.table.TotalPOIGasUsed.Put(key, gasBytes)
	if err != nil {
		s.Log.Crit("Failed to set key", "err", err)
	}
}

// AddPOIGasUsed add gas used to POI period
func (s *Store) AddPOIGasUsed(poiPeriod uint64, gas uint64) {
	oldGas := s.GetPOIGasUsed(poiPeriod)
	s.SetPOIGasUsed(poiPeriod, gas + oldGas)
}

// GetPOIGasUsed get gas used for POI period
func (s *Store) GetPOIGasUsed(poiPeriod uint64) uint64 {
	key := bigendian.Int64ToBytes(poiPeriod)

	gasBytes, err := s.table.TotalPOIGasUsed.Get(key)
	if err != nil {
		s.Log.Crit("Failed to get key", "err", err)
	}

	gas := bigendian.BytesToInt64(gasBytes)

	return gas
}

// SetValidatorPOI save POI value for validator address
func (s *Store) SetValidatorPOI(addr common.Address, poi uint64) {
	poiBytes := bigendian.Int64ToBytes(poi)
	err := s.table.ValidatorPOIScore.Put(addr.Bytes(), poiBytes)
	if err != nil {
		s.Log.Crit("Failed to set key", "err", err)
	}
}

// GetValidatorPOI get POI value for validator address
func (s *Store) GetValidatorPOI(addr common.Address) uint64 {
	poiBytes, err := s.table.ValidatorPOIScore.Get(addr.Bytes())
	if err != nil {
		s.Log.Crit("Failed to set key", "err", err)
	}

	poi := bigendian.BytesToInt64(poiBytes)

	return poi
}

// TODO: SaveValidatorsSnapshotGroup save sorted validators top 30 as validators snapshot group
func (s *Store) SaveValidatorsSnapshotGroup() error {
	/*
		write snapshot into the contract storage
			for each V from the validators group
				write V into the snapshot (including validating power, with active scores)
	*/





	return nil
}

// TODO: SwitchValidatorsSnapshotGroup switch current snapshot to new snapshot group
func (s *Store) SwitchValidatorsSnapshotGroup() {
	/*
	choose new validators group. currently, not specified how exactly new group is calculated
	 */




}
