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

// Params is used to group parameters for various networks such as the main
// network and test networks.
// type Params struct {
// 	*chaincfg.Params
// 	RPCClientPort string
// 	RPCServerPort string
// }

// // MainNetParams contains parameters specific running btcwallet and
// // btcd on the main network (wire.MainNet).
// var MainNetParams = Params{
// 	Params:        &chaincfg.MainNetParams,
// 	RPCClientPort: "8334",
// 	RPCServerPort: "8332",
// }

// // TestNet3Params contains parameters specific running btcwallet and
// // btcd on the test network (version 3) (wire.TestNet3).
// var TestNet3Params = Params{
// 	Params:        &chaincfg.TestNet3Params,
// 	RPCClientPort: "18334",
// 	RPCServerPort: "18332",
// }

// // SimNetParams contains parameters specific to the simulation test network
// // (wire.SimNet).
// var SimNetParams = Params{
// 	Params:        &chaincfg.SimNetParams,
// 	RPCClientPort: "18556",
// 	RPCServerPort: "18554",
// }

// // SigNetParams contains parameters specific to the signet test network
// // (wire.SigNet).
// var SigNetParams = Params{
// 	Params:        &chaincfg.SigNetParams,
// 	RPCClientPort: "38334",
// 	RPCServerPort: "38332",
// }

// // RegressionNetParams contains parameters specific to the regression test
// // network (wire.RegressionNet).
// var RegressionNetParams = Params{
// 	Params:        &chaincfg.RegressionNetParams,
// 	RPCClientPort: "18334",
// 	RPCServerPort: "18332",
// }

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
	}
}

var NSimnetParams = &ChainParams{}
var NMainnetParams = &ChainParams{}
