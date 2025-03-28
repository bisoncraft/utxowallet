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

type ChainParams struct {
	Name                     string
	Net                      wire.BitcoinNet
	GenesisBlock             *wire.MsgBlock
	GenesisHash              *chainhash.Hash
	TargetTimespan           time.Duration
	TargetTimePerBlock       time.Duration
	RetargetAdjustmentFactor int64
	Checkpoints              []chaincfg.Checkpoint
	PowLimit                 *big.Int
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

	// CheckPoW is a function that will check the proof-of-work validity for a
	// block header. If CheckPoW is nil, the standard Bitcoin protocol is used.
	CheckPoW func(*wire.BlockHeader) error
	// MaxSatoshi varies between assets.
	MaxSatoshi int64
}

func (c *ChainParams) BTCDParams() *chaincfg.Params {
	return &chaincfg.Params{
		Name:                     c.Name,
		Net:                      c.Net,
		GenesisBlock:             c.GenesisBlock,
		GenesisHash:              c.GenesisHash,
		TargetTimespan:           c.TargetTimespan,
		TargetTimePerBlock:       c.TargetTimePerBlock,
		RetargetAdjustmentFactor: c.RetargetAdjustmentFactor,
		Checkpoints:              c.Checkpoints,
		PowLimit:                 c.PowLimit,
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
	}
}
