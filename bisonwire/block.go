package bisonwire

import (
	"fmt"
	"io"

	"github.com/btcsuite/btcd/wire"
)

// Block is a wire.MsgBlock, but with *bisonwire.Tx instead of wire.MsgTx.
type Block struct {
	Chain        string
	Header       wire.BlockHeader
	Transactions []*Tx
}

// MsgBlock generates a *wire.MsgBlock from the *Block.
func (b *Block) MsgBlock() *wire.MsgBlock {
	msgBlock := &wire.MsgBlock{
		Header:       b.Header,
		Transactions: make([]*wire.MsgTx, len(b.Transactions)),
	}
	for i, tx := range b.Transactions {
		msgBlock.Transactions[i] = &tx.MsgTx
	}
	return msgBlock
}

// BlockFromMsgBlock generates a *Block, with a *wire.MsgBlock as it's base. The
// wire.MsgTx's are converted to *Tx. This is only used in tests for now, So we
// can ignore the additional Tx fields that aren't contained in *wire.MsgBlock.
func BlockFromMsgBlock(chain string, msgBlock *wire.MsgBlock) *Block {
	blk := &Block{
		Chain:        chain,
		Header:       msgBlock.Header,
		Transactions: make([]*Tx, len(msgBlock.Transactions)),
	}
	for i, msgTx := range msgBlock.Transactions {
		blk.Transactions[i] = &Tx{
			MsgTx: *msgTx,
			Chain: chain,
		}
	}
	return blk
}

// BtcDecode decodes the serialized block based on the Block's Chain.
func (b *Block) BtcDecode(r io.Reader, pver uint32, enc wire.MessageEncoding) error {
	switch b.Chain {
	case "btc":
		var msgBlock wire.MsgBlock
		if err := msgBlock.BtcDecode(r, pver, enc); err != nil {
			return fmt.Errorf("error decoding Bitcoin block: %w", err)
		}
		b.Header = msgBlock.Header
		b.Transactions = make([]*Tx, len(msgBlock.Transactions))
		for i, msgTx := range msgBlock.Transactions {
			b.Transactions[i] = &Tx{
				MsgTx: *msgTx,
				Chain: b.Chain,
			}
		}
	case "ltc":
		blk, err := deserializeLitecoinBlock(r)
		if err != nil {
			return fmt.Errorf("error decoding Bitcoin block: %w", err)
		}
		b.Header = blk.Header
		b.Transactions = blk.Transactions
	default:
		return fmt.Errorf("unknown chain %q specified for block decoding", b.Chain)
	}
	return nil
}

func (b *Block) BtcEncode(r io.Writer, pver uint32, enc wire.MessageEncoding) error {
	switch b.Chain {
	case "btc":
		return b.MsgBlock().BtcEncode(r, pver, enc)
	case "ltc":
		panic("bisonwire block encoding not implemented for Litecoin") // I don't think we ever use this
	default:
		return fmt.Errorf("unknown chain %q specified for block decoding", b.Chain)
	}
}

func (b *Block) Command() string {
	return wire.CmdBlock
}

func (b *Block) MaxPayloadLength(uint32) uint32 {
	return wire.MaxBlockPayload
}

type BlockWithHeight struct {
	Block  *Block
	Height uint32
}
