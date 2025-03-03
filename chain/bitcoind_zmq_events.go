package chain

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
)

// getTxSpendingPrevOut makes an RPC call to `gettxspendingprevout` and returns
// the result.
func getTxSpendingPrevOut(op wire.OutPoint,
	client *rpcclient.Client) (chainhash.Hash, bool) {

	prevoutResps, err := client.GetTxSpendingPrevOut([]wire.OutPoint{op})
	if err != nil {
		return chainhash.Hash{}, false
	}

	// We should only get a single item back since we only requested with a
	// single item.
	if len(prevoutResps) != 1 {
		return chainhash.Hash{}, false
	}

	result := prevoutResps[0]

	// If the "spendingtxid" field is empty, then the utxo has no spend in
	// the mempool at the moment.
	if result.SpendingTxid == "" {
		return chainhash.Hash{}, false
	}

	spendHash, err := chainhash.NewHashFromStr(result.SpendingTxid)
	if err != nil {
		return chainhash.Hash{}, false
	}

	return *spendHash, true
}
