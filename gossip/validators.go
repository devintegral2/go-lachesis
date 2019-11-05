package gossip

import (
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/Fantom-foundation/go-lachesis/common/bigendian"
)

const (
	PoiPeriodDuration = 2 * 24 * time.Hour
)

func PoiPeriod(t int64) uint64 {
	return uint64(t / int64(PoiPeriodDuration))
}

func (s *Store) CalcValidatorsPOI(validator, delegator common.Address, poiPeriod uint64) {
	vGasUsed := s.GetAddressGasUsed(validator)
	dGasUsed := s.GetAddressGasUsed(delegator)

	vGasUsed += dGasUsed
	s.SetAddressGasUsed(validator, vGasUsed)

	poi := uint64((vGasUsed * 1000000) / s.GetPOIGasUsed(poiPeriod))
	s.SetValidatorPOI(validator, poi)
}

func (s *Store) GetAddressGasUsed(addr common.Address) uint64 {
	gasBytes, err := s.table.AddressGasUsed.Get(addr.Bytes())
	if err != nil {
		s.Log.Crit("Failed to get key", "err", err)
	}

	gas := bigendian.BytesToInt64(gasBytes)

	return gas
}

func (s *Store) SetAddressGasUsed(addr common.Address, gas uint64) {
	gasBytes := bigendian.Int64ToBytes(gas)

	err := s.table.AddressGasUsed.Put(addr.Bytes(), gasBytes)
	if err != nil {
		s.Log.Crit("Failed to set key", "err", err)
	}
}

func (s *Store) GetAddressLastTrxTime(addr common.Address) uint64 {
	gasBytes, err := s.table.AddressLastTrxTime.Get(addr.Bytes())
	if err != nil {
		s.Log.Crit("Failed to get key", "err", err)
	}

	gas := bigendian.BytesToInt64(gasBytes)

	return gas
}

func (s *Store) SetAddressLastTrxTime(addr common.Address, gas uint64) {
	gasBytes := bigendian.Int64ToBytes(gas)

	err := s.table.AddressLastTrxTime.Put(addr.Bytes(), gasBytes)
	if err != nil {
		s.Log.Crit("Failed to set key", "err", err)
	}
}

func (s *Store) SetPOIGasUsed(poiPeriod uint64, gas uint64) {
	key := bigendian.Int64ToBytes(poiPeriod)
	gasBytes := bigendian.Int64ToBytes(gas)

	err := s.table.TotalPOIGasUsed.Put(key, gasBytes)
	if err != nil {
		s.Log.Crit("Failed to set key", "err", err)
	}
}

func (s *Store) AddPOIGasUsed(poiPeriod uint64, gas uint64) {
	oldGas := s.GetPOIGasUsed(poiPeriod)
	s.SetPOIGasUsed(poiPeriod, gas + oldGas)
}

func (s *Store) GetPOIGasUsed(poiPeriod uint64) uint64 {
	key := bigendian.Int64ToBytes(poiPeriod)

	gasBytes, err := s.table.TotalPOIGasUsed.Get(key)
	if err != nil {
		s.Log.Crit("Failed to get key", "err", err)
	}

	gas := bigendian.BytesToInt64(gasBytes)

	return gas
}

func (s *Store) SetValidatorPOI(addr common.Address, poi uint64) {
	poiBytes := bigendian.Int64ToBytes(poi)
	err := s.table.ValidatorPOIScore.Put(addr.Bytes(), poiBytes)
	if err != nil {
		s.Log.Crit("Failed to set key", "err", err)
	}
}

func (s *Store) GetValidatorPOI(addr common.Address) uint64 {
	poiBytes, err := s.table.ValidatorPOIScore.Get(addr.Bytes())
	if err != nil {
		s.Log.Crit("Failed to set key", "err", err)
	}

	poi := bigendian.BytesToInt64(poiBytes)

	return poi
}
