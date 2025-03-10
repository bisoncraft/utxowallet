package chain

import (
	"github.com/bisoncraft/utxowallet/spv"
	"github.com/bisoncraft/utxowallet/spv/banman"
	"github.com/bisoncraft/utxowallet/spv/headerfs"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/gcs"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

// NeutrinoChainService is an interface that encapsulates all the public
// methods of an *spv.ChainService
type NeutrinoChainService interface {
	Start() error
	GetBlock(chainhash.Hash, ...spv.QueryOption) (*btcutil.Block, error)
	GetBlockHeight(*chainhash.Hash) (int32, error)
	BestBlock() (*headerfs.BlockStamp, error)
	GetBlockHash(int64) (*chainhash.Hash, error)
	GetBlockHeader(*chainhash.Hash) (*wire.BlockHeader, error)
	IsCurrent() bool
	SendTransaction(*wire.MsgTx) error
	GetCFilter(chainhash.Hash, wire.FilterType,
		...spv.QueryOption) (*gcs.Filter, error)
	GetUtxo(...spv.RescanOption) (*spv.SpendReport, error)
	BanPeer(string, banman.Reason) error
	IsBanned(addr string) bool
	AddPeer(*spv.ServerPeer)
	AddBytesSent(uint64)
	AddBytesReceived(uint64)
	NetTotals() (uint64, uint64)
	UpdatePeerHeights(*chainhash.Hash, int32, *spv.ServerPeer)
	ChainParams() chaincfg.Params
	Stop() error
	PeerByAddr(string) *spv.ServerPeer
}

var _ NeutrinoChainService = (*spv.ChainService)(nil)
