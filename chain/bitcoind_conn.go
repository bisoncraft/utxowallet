package chain

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
)

const (
	// rawBlockZMQCommand is the command used to receive raw block
	// notifications from bitcoind through ZMQ.
	rawBlockZMQCommand = "rawblock"

	// rawTxZMQCommand is the command used to receive raw transaction
	// notifications from bitcoind through ZMQ.
	rawTxZMQCommand = "rawtx"

	// maxRawBlockSize is the maximum size in bytes for a raw block received
	// from bitcoind through ZMQ.
	maxRawBlockSize = 4e6

	// maxRawTxSize is the maximum size in bytes for a raw transaction
	// received from bitcoind through ZMQ.
	maxRawTxSize = maxRawBlockSize

	// seqNumLen is the length of the sequence number of a message sent from
	// bitcoind through ZMQ.
	seqNumLen = 4

	// errBlockPrunedStr is the error message returned by bitcoind upon
	// calling GetBlock on a pruned block.
	errBlockPrunedStr = "Block not available (pruned data)"

	// errStillLoadingCode is the error code returned when an RPC request
	// is made but bitcoind is still in the process of loading or verifying
	// blocks.
	errStillLoadingCode = "-28"

	// bitcoindStartTimeout is the time we wait for bitcoind to finish
	// loading and verifying blocks and become ready to serve RPC requests.
	bitcoindStartTimeout = 30 * time.Second
)

// ErrBitcoindStartTimeout is returned when the bitcoind daemon fails to load
// and verify blocks under 30s during startup.
var ErrBitcoindStartTimeout = errors.New("bitcoind start timeout")

// Dialer represents a way to dial Bitcoin peers. If the chain backend is
// running over Tor, this must support dialing peers over Tor as well.
type Dialer = func(string) (net.Conn, error)

// getBlockHashDuringStartup is used to call the getblockhash RPC during
// startup. It catches the case where bitcoind is still in the process of
// loading blocks, which returns the following error,
// - "-28: Loading block index..."
// - "-28: Verifying blocks..."
// In this case we'd retry every second until we time out after 30 seconds.
func getBlockHashDuringStartup(
	client *rpcclient.Client) (*chainhash.Hash, error) {

	hash, err := client.GetBlockHash(0)

	// Exit early if there's no error.
	if err == nil {
		return hash, nil
	}

	// If the error doesn't start with "-28", it's an unexpected error so
	// we exit with it.
	if !strings.Contains(err.Error(), errStillLoadingCode) {
		return nil, err
	}

	// Starts the timeout ticker(30s).
	timeout := time.After(bitcoindStartTimeout)

	// Otherwise, we'd retry calling getblockhash or time out after 30s.
	for {
		select {
		case <-timeout:
			return nil, ErrBitcoindStartTimeout

		// Retry every second.
		case <-time.After(1 * time.Second):
			hash, err = client.GetBlockHash(0)
			// If there's no error, we return the hash.
			if err == nil {
				return hash, nil
			}

			// Otherwise, retry until we time out. We also check if
			// the error returned here is still expected.
			if !strings.Contains(err.Error(), errStillLoadingCode) {
				return nil, err
			}
		}
	}
}

// getCurrentNet returns the network on which the bitcoind node is running.
func getCurrentNet(client *rpcclient.Client) (wire.BitcoinNet, error) {
	hash, err := getBlockHashDuringStartup(client)
	if err != nil {
		return 0, err
	}

	switch *hash {
	case *chaincfg.TestNet3Params.GenesisHash:
		return chaincfg.TestNet3Params.Net, nil
	case *chaincfg.RegressionNetParams.GenesisHash:
		return chaincfg.RegressionNetParams.Net, nil
	case *chaincfg.SigNetParams.GenesisHash:
		return chaincfg.SigNetParams.Net, nil
	case *chaincfg.MainNetParams.GenesisHash:
		return chaincfg.MainNetParams.Net, nil
	default:
		return 0, fmt.Errorf("unknown network with genesis hash %v", hash)
	}
}

// isBlockPrunedErr determines if the error returned by the GetBlock RPC
// corresponds to the requested block being pruned.
func isBlockPrunedErr(err error) bool {
	rpcErr, ok := err.(*btcjson.RPCError)
	return ok && rpcErr.Code == btcjson.ErrRPCMisc &&
		rpcErr.Message == errBlockPrunedStr
}

// isASCII is a helper method that checks whether all bytes in `data` would be
// printable ASCII characters if interpreted as a string.
func isASCII(s string) bool {
	for _, c := range s {
		if c < 32 || c > 126 {
			return false
		}
	}
	return true
}
