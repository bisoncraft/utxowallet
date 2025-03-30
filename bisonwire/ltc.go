package bisonwire

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"lukechampine.com/blake3"
)

func litcoinTxHash(tx *Tx) chainhash.Hash {
	// A pure-MW tx can only be in mempool or the EB, not the canonical block.
	if len(tx.Kern0) > 0 && len(tx.TxIn) == 0 && len(tx.TxOut) == 0 {
		// CTransaction::ComputeHash in src/primitives/transaction.cpp.
		// Fortunately also a 32 byte hash so we can use chainhash.Hash.
		return blake3.Sum256(tx.Kern0)
	}
	return tx.MsgTx.TxHash()
}

func deserializeLitecoinTx(r io.Reader, tx *Tx) error {
	msgTx := &tx.MsgTx
	dec := newDecoder(r)

	version, err := dec.readUint32()
	if err != nil {
		return err
	}
	// if version != 0 {
	// 	return nil, fmt.Errorf("only tx version 0 supported, got %d", version)
	// }
	msgTx.Version = int32(version)

	count, err := dec.readCompactSize()
	if err != nil {
		return err
	}

	// A count of zero (meaning no TxIn's to the uninitiated) means that the
	// value is a TxFlagMarker, and hence indicates the presence of a flag.
	var flag [1]byte
	if count == 0 {
		// The count varint was in fact the flag marker byte. Next, we need to
		// read the flag value, which is a single byte.
		if _, err = io.ReadFull(r, flag[:]); err != nil {
			return err
		}

		// Flag bits 0 or 3 must be set.
		if flag[0]&0b1001 == 0 {
			return fmt.Errorf("witness tx but flag byte is %x", flag)
		}

		// With the Segregated Witness specific fields decoded, we can
		// now read in the actual txin count.
		count, err = dec.readCompactSize()
		if err != nil {
			return err
		}
	}

	if count > maxTxInPerMessage {
		return fmt.Errorf("too many transaction inputs to fit into "+
			"max message size [count %d, max %d]", count, maxTxInPerMessage)
	}

	msgTx.TxIn = make([]*wire.TxIn, count)
	for i := range msgTx.TxIn {
		txIn := &wire.TxIn{}
		err = dec.readTxIn(txIn)
		if err != nil {
			return err
		}
		msgTx.TxIn[i] = txIn
	}

	count, err = dec.readCompactSize()
	if err != nil {
		return err
	}
	if count > maxTxOutPerMessage {
		return fmt.Errorf("too many transactions outputs to fit into "+
			"max message size [count %d, max %d]", count, maxTxOutPerMessage)
	}

	msgTx.TxOut = make([]*wire.TxOut, count)
	for i := range msgTx.TxOut {
		txOut := &wire.TxOut{}
		err = dec.readTxOut(txOut)
		if err != nil {
			return err
		}
		msgTx.TxOut[i] = txOut
	}

	if flag[0]&0x01 != 0 {
		for _, txIn := range msgTx.TxIn {
			witCount, err := dec.readCompactSize()
			if err != nil {
				return err
			}
			if witCount > maxWitnessItemsPerInput {
				return fmt.Errorf("too many witness items: %d > max %d",
					witCount, maxWitnessItemsPerInput)
			}
			txIn.Witness = make(wire.TxWitness, witCount)
			for j := range txIn.Witness {
				txIn.Witness[j], err = wire.ReadVarBytes(r, pver,
					maxWitnessItemSize, "script witness item")
				if err != nil {
					return err
				}
			}
		}
	}

	// check for a MW tx based on flag 0x08
	// src/primitives/transaction.h - class CTransaction
	// Serialized like normal tx except with an optional MWEB::Tx after outputs
	// and before locktime.
	if flag[0]&0x08 != 0 {
		tx.Kern0, tx.IsHogEx, err = dec.readMWTX()
		if err != nil {
			return err
		}
		if tx.IsHogEx && len(msgTx.TxOut) == 0 {
			return errors.New("no outputs on HogEx txn")
		}
	}

	msgTx.LockTime, err = dec.readUint32()
	return err
}

// Block

const (
	// mwebVer is the bit of the block header's version that indicates the
	// presence of a MWEB.
	mwebVer = 0x20000000 // 1 << 29
)

func parseMWEB(blk io.Reader) error {
	dec := newDecoder(blk)
	// src/mweb/mweb_models.h - struct Block
	// "A convenience wrapper around a possibly-null extension block.""
	// OptionalPtr around a mw::Block. Read the option byte:
	hasMWEB, err := dec.readByte()
	if err != nil {
		return fmt.Errorf("failed to check MWEB option byte: %w", err)
	}
	if hasMWEB == 0 {
		return nil
	}

	// src/libmw/include/mw/models/block/Block.h - class Block
	// (1) Header and (2) TxBody

	// src/libmw/include/mw/models/block/Header.h - class Header
	// height
	if _, err = dec.readVLQ(); err != nil {
		return fmt.Errorf("failed to decode MWEB height: %w", err)
	}

	// 3x Hash + 2x BlindingFactor
	if err = dec.discardBytes(32*3 + 32*2); err != nil {
		return fmt.Errorf("failed to decode MWEB junk: %w", err)
	}

	// Number of TXOs: outputMMRSize
	if _, err = dec.readVLQ(); err != nil {
		return fmt.Errorf("failed to decode TXO count: %w", err)
	}

	// Number of kernels: kernelMMRSize
	if _, err = dec.readVLQ(); err != nil {
		return fmt.Errorf("failed to decode kernel count: %w", err)
	}

	// TxBody
	_, err = dec.readMWTXBody()
	if err != nil {
		return fmt.Errorf("failed to decode MWEB tx: %w", err)
	}
	// if len(kern0) > 0 {
	// 	mwebTxID := chainhash.Hash(blake3.Sum256(kern0))
	// 	fmt.Println(mwebTxID.String())
	// }

	return nil
}

// DeserializeBlock decodes the bytes of a serialized Litecoin block. This
// function exists because MWEB changes both the block and transaction
// serializations. Blocks may have a MW "extension block" for "peg-out"
// transactions, and this EB is after all the transactions in the regular LTC
// block. After the canonical transactions in the regular block, there may be
// zero or more "peg-in" transactions followed by one integration transaction
// (also known as a HogEx transaction), all still in the regular LTC block. The
// peg-in txns decode correctly, but the integration tx is a special transaction
// with the witness tx flag with bit 3 set (8), which prevents correct
// wire.MsgTx deserialization.
// Refs:
// https://github.com/litecoin-project/lips/blob/master/lip-0002.mediawiki#PegOut_Transactions
// https://github.com/litecoin-project/lips/blob/master/lip-0003.mediawiki#Specification
// https://github.com/litecoin-project/litecoin/commit/9d1f530a5fa6d16871fdcc3b506be42b593d3ce4
// https://github.com/litecoin-project/litecoin/commit/8c82032f45e644f413ec5c91e121a31c993aa831
// (src/libmw/include/mw/models/tx/Transaction.h for the `mweb_tx` field of the
// `CTransaction` in the "primitives" commit).
func deserializeLitecoinBlock(r io.Reader) (*Block, error) {
	// Block header
	hdr := &wire.BlockHeader{}
	err := hdr.Deserialize(r)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize block header: %w", err)
	}

	// This block's transactions
	txnCount, err := wire.ReadVarInt(r, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transaction count: %w", err)
	}

	// We can only decode the canonical txns, not the mw peg-in txs in the EB.
	var hasHogEx bool
	txns := make([]*Tx, 0, int(txnCount))
	for i := 0; i < cap(txns); i++ {
		tx := &Tx{Chain: "ltc"}
		if err := deserializeLitecoinTx(r, tx); err != nil {
			return nil, fmt.Errorf("failed to deserialize transaction %d of %d in block %v: %w",
				i+1, txnCount, hdr.BlockHash(), err)
		}
		txns = append(txns, tx) // txns = append(txns, msgTx)
		hasHogEx = tx.IsHogEx   // hogex is the last txn
	}

	// The mwebVer mask indicates it may contain a MWEB after a HogEx.
	// src/primitives/block.h: SERIALIZE_NO_MWEB
	if hdr.Version&mwebVer != 0 && hasHogEx {
		if err = parseMWEB(r); err != nil {
			return nil, err
		}
	}

	return &Block{
		Header:       *hdr,
		Transactions: txns,
	}, nil
}

// deserializeLitecoinBlockBytes wraps DeserializeBlock using bytes.NewReader for
// convenience.
func deserializeLitecoinBlockBytes(blk []byte) (*Block, error) {
	return deserializeLitecoinBlock(bytes.NewReader(blk))
}
