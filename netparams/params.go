// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package netparams

import (
	"math/big"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

// ChainParams are an extended version of btcd/chaincfg.Params that omit some
// unnecessary fields and add some fields necessary for multi-asset needs.
type ChainParams struct {
	Chain                    string
	Name                     string
	Net                      wire.BitcoinNet
	GenesisBlock             *wire.MsgBlock
	GenesisHash              *chainhash.Hash
	TargetTimespan           time.Duration
	TargetTimePerBlock       time.Duration
	RetargetAdjustmentFactor int64
	ReduceMinDifficulty      bool
	MinDiffReductionTime     time.Duration
	Checkpoints              []chaincfg.Checkpoint
	PowLimit                 *big.Int
	PowLimitBits             uint32
	PoWNoRetargeting         bool
	DNSSeeds                 []chaincfg.DNSSeed
	DefaultPort              string

	// Human-readable part for Bech32 encoded segwit addresses, as defined
	// in BIP 173.
	Bech32HRPSegwit string

	// Address encoding magics
	PubKeyHashAddrID        byte // First byte of a P2PKH address
	ScriptHashAddrID        byte // First byte of a P2SH address
	PrivateKeyID            byte // First byte of a WIF private key
	WitnessPubKeyHashAddrID byte // First byte of a P2WPKH address
	WitnessScriptHashAddrID byte // First byte of a P2WSH address

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID [4]byte
	HDPublicKeyID  [4]byte

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType uint32

	BIP0034Height int32
	BIP0065Height int32
	BIP0066Height int32

	CoinbaseMaturity uint16

	// CheckPoW is a function that will check the proof-of-work validity for a
	// block header. If CheckPoW is nil, the standard Bitcoin protocol is used.
	CheckPoW func(*wire.BlockHeader) error
	// MaxSatoshi varies between assets.
	MaxSatoshi int64

	btcdParams *chaincfg.Params
}

// BTCDParams are the btcd.chaincfg.Params that correspond the ChainParams.
func (c *ChainParams) BTCDParams() *chaincfg.Params {
	if c.btcdParams != nil {
		return c.btcdParams
	}
	c.btcdParams = &chaincfg.Params{
		Name:                     c.Name,
		Net:                      c.Net,
		GenesisBlock:             c.GenesisBlock,
		GenesisHash:              c.GenesisHash,
		TargetTimespan:           c.TargetTimespan,
		TargetTimePerBlock:       c.TargetTimePerBlock,
		RetargetAdjustmentFactor: c.RetargetAdjustmentFactor,
		ReduceMinDifficulty:      c.ReduceMinDifficulty,
		MinDiffReductionTime:     c.MinDiffReductionTime,
		Checkpoints:              c.Checkpoints,
		PowLimit:                 c.PowLimit,
		PowLimitBits:             c.PowLimitBits,
		PoWNoRetargeting:         c.PoWNoRetargeting,
		DNSSeeds:                 c.DNSSeeds,
		DefaultPort:              c.DefaultPort,
		Bech32HRPSegwit:          c.Bech32HRPSegwit,
		PubKeyHashAddrID:         c.PubKeyHashAddrID,
		ScriptHashAddrID:         c.ScriptHashAddrID,
		PrivateKeyID:             c.PrivateKeyID,
		WitnessPubKeyHashAddrID:  c.WitnessPubKeyHashAddrID,
		WitnessScriptHashAddrID:  c.WitnessScriptHashAddrID,
		HDPrivateKeyID:           c.HDPrivateKeyID,
		HDPublicKeyID:            c.HDPublicKeyID,
		HDCoinType:               c.HDCoinType,
		BIP0034Height:            c.BIP0034Height,
		BIP0065Height:            c.BIP0065Height,
		BIP0066Height:            c.BIP0066Height,
		CoinbaseMaturity:         c.CoinbaseMaturity,
	}
	return c.btcdParams
}
