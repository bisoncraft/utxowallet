package bisonwire

import (
	"fmt"
	"io"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

type Tx struct {
	wire.MsgTx
	Chain   string
	IsHogEx bool
	Kern0   []byte
}

func (tx *Tx) Deserialize(r io.Reader) error {
	switch tx.Chain {
	case "btc":
		return tx.MsgTx.Deserialize(r)
	case "ltc":
		return deserializeLitcoinTx(r, tx)
	default:
		return fmt.Errorf("unknown chain %q specified for bisonwire tx deserialization", tx.Chain)
	}
}

func (tx *Tx) TxHash() chainhash.Hash {
	switch tx.Chain {
	case "btc":
		return tx.MsgTx.TxHash()
	case "ltc":
		return litcoinTxHash(tx)
	default:
		panic(fmt.Sprintf("unknown chain %q specified for bisonwire tx hash", tx.Chain))
	}
}
