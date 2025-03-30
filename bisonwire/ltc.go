package bisonwire

import (
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

func deserializeLitcoinTx(r io.Reader, tx *Tx) error {
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
